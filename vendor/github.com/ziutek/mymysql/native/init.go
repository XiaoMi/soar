package native

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"log"

	"github.com/ziutek/mymysql/mysql"
)

func (my *Conn) init() {
	my.seq = 0 // Reset sequence number, mainly for reconnect
	if my.Debug {
		log.Printf("[%2d ->] Init packet:", my.seq)
	}
	pr := my.newPktReader()

	my.info.prot_ver = pr.readByte()
	my.info.serv_ver = pr.readNTB()
	my.info.thr_id = pr.readU32()
	pr.readFull(my.info.scramble[0:8])
	pr.skipN(1)
	my.info.caps = uint32(pr.readU16()) // lower two bytes
	my.info.lang = pr.readByte()
	my.status = mysql.ConnStatus(pr.readU16())
	my.info.caps = uint32(pr.readU16())<<16 | my.info.caps // upper two bytes
	pr.skipN(11)
	if my.info.caps&_CLIENT_PROTOCOL_41 != 0 {
		pr.readFull(my.info.scramble[8:])
	}
	pr.skipN(1) // reserved (all [00])
	if my.info.caps&_CLIENT_PLUGIN_AUTH != 0 {
		my.info.plugin = pr.readNTB()
	}
	pr.skipAll() // Skip other information
	if my.Debug {
		log.Printf(tab8s+"ProtVer=%d, ServVer=\"%s\" Status=0x%x",
			my.info.prot_ver, my.info.serv_ver, my.status,
		)
	}
	if my.info.caps&_CLIENT_PROTOCOL_41 == 0 {
		panic(mysql.ErrOldProtocol)
	}
}

func (my *Conn) auth() {
	if my.Debug {
		log.Printf("[%2d <-] Authentication packet", my.seq)
	}
	flags := uint32(
		_CLIENT_PROTOCOL_41 |
			_CLIENT_LONG_PASSWORD |
			_CLIENT_LONG_FLAG |
			_CLIENT_TRANSACTIONS |
			_CLIENT_SECURE_CONN |
			_CLIENT_LOCAL_FILES |
			_CLIENT_MULTI_STATEMENTS |
			_CLIENT_MULTI_RESULTS)
	// Reset flags not supported by server
	flags &= uint32(my.info.caps) | 0xffff0000
	if my.plugin != string(my.info.plugin) {
		my.plugin = string(my.info.plugin)
	}
	var scrPasswd []byte
	switch my.plugin {
	case "caching_sha2_password":
		flags |= _CLIENT_PLUGIN_AUTH
		scrPasswd = encryptedSHA256Passwd(my.passwd, my.info.scramble[:])
	case "mysql_old_password":
		my.oldPasswd()
		return
	default:
		// mysql_native_password by default
		scrPasswd = encryptedPasswd(my.passwd, my.info.scramble[:])
	}

	// encode length of the auth plugin data
	var authRespLEIBuf [9]byte
	authRespLEI := appendLengthEncodedInteger(authRespLEIBuf[:0], uint64(len(scrPasswd)))
	if len(authRespLEI) > 1 {
		// if the length can not be written in 1 byte, it must be written as a
		// length encoded integer
		flags |= _CLIENT_PLUGIN_AUTH_LENENC_CLIENT_DATA
	}

	pay_len := 4 + 4 + 1 + 23 + len(my.user) + 1 + len(authRespLEI) + len(scrPasswd) + 21 + 1

	if len(my.dbname) > 0 {
		pay_len += len(my.dbname) + 1
		flags |= _CLIENT_CONNECT_WITH_DB
	}
	pw := my.newPktWriter(pay_len)
	pw.writeU32(flags)
	pw.writeU32(uint32(my.max_pkt_size))
	pw.writeByte(my.info.lang)   // Charset number
	pw.writeZeros(23)            // Filler
	pw.writeNTB([]byte(my.user)) // Username
	pw.writeBin(scrPasswd)       // Encrypted password

	// write database name
	if len(my.dbname) > 0 {
		pw.writeNTB([]byte(my.dbname))
	}

	// write plugin name
	if my.plugin != "" {
		pw.writeNTB([]byte(my.plugin))
	} else {
		pw.writeNTB([]byte("mysql_native_password"))
	}
	return
}

func (my *Conn) authResponse() {
	// Read Result Packet
	authData, newPlugin := my.getAuthResult()

	// handle auth plugin switch, if requested
	if newPlugin != "" {
		var scrPasswd []byte
		if len(authData) >= 20 {
			// old_password's len(authData) == 0
			copy(my.info.scramble[:], authData[:20])
		}
		my.info.plugin = []byte(newPlugin)
		my.plugin = newPlugin
		switch my.plugin {
		case "caching_sha2_password":
			scrPasswd = encryptedSHA256Passwd(my.passwd, my.info.scramble[:])
		case "mysql_old_password":
			scrPasswd = encryptedOldPassword(my.passwd, my.info.scramble[:])
			// append \0 after old_password
			scrPasswd = append(scrPasswd, 0)
		case "sha256_password":
			// request public key from server
			scrPasswd = []byte{1}
		default: // mysql_native_password
			scrPasswd = encryptedPasswd(my.passwd, my.info.scramble[:])
		}
		my.writeAuthSwitchPacket(scrPasswd)

		// Read Result Packet
		authData, newPlugin = my.getAuthResult()

		// Do not allow to change the auth plugin more than once
		if newPlugin != "" {
			return
		}
	}

	switch my.plugin {

	// https://insidemysql.com/preparing-your-community-connector-for-mysql-8-part-2-sha256/
	case "caching_sha2_password":
		switch len(authData) {
		case 0:
			return // auth successful
		case 1:
			switch authData[0] {
			case 3: // cachingSha2PasswordFastAuthSuccess
				my.getResult(nil, nil)

			case 4: // cachingSha2PasswordPerformFullAuthentication
				// request public key from server
				pw := my.newPktWriter(1)
				pw.writeByte(2)

				// parse public key
				pr := my.newPktReader()
				pr.skipN(1)
				data := pr.readAll()
				block, _ := pem.Decode(data)
				pkix, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					panic(mysql.ErrAuthentication)
				}
				pubKey := pkix.(*rsa.PublicKey)

				// send encrypted password
				my.sendEncryptedPassword(my.info.scramble[:], pubKey)
				my.getResult(nil, nil)
			}
		}
	case "sha256_password":
		switch len(authData) {
		case 0:
			return // auth successful
		default:
			// parse public key
			block, _ := pem.Decode(authData)
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				panic(mysql.ErrAuthentication)
			}

			// send encrypted password
			my.sendEncryptedPassword(my.info.scramble[:], pub.(*rsa.PublicKey))
			my.getResult(nil, nil)
		}
	}
	return
}

// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::AuthSwitchResponse
func (my *Conn) writeAuthSwitchPacket(scrPasswd []byte) {
	pw := my.newPktWriter(len(scrPasswd))
	pw.write(scrPasswd) // Encrypted password
	return
}

func (my *Conn) sendEncryptedPassword(seed []byte, pub *rsa.PublicKey) {
	enc, err := encryptPassword(my.passwd, seed, pub)
	if err != nil {
		panic(mysql.ErrAuthentication)
	}
	my.writeAuthSwitchPacket(enc)
}

func encryptPassword(password string, seed []byte, pub *rsa.PublicKey) ([]byte, error) {
	plain := make([]byte, len(password)+1)
	copy(plain, password)
	for i := range plain {
		j := i % len(seed)
		plain[i] ^= seed[j]
	}
	sha1 := sha1.New()
	return rsa.EncryptOAEP(sha1, rand.Reader, pub, plain, nil)
}

func (my *Conn) oldPasswd() {
	if my.Debug {
		log.Printf("[%2d <-] Password packet", my.seq)
	}
	scrPasswd := encryptedOldPassword(my.passwd, my.info.scramble[:])
	pw := my.newPktWriter(len(scrPasswd) + 1)
	pw.write(scrPasswd)
	pw.writeByte(0)
}
