Package uniuri
=====================

[![Build Status](https://travis-ci.org/dchest/uniuri.svg)](https://travis-ci.org/dchest/uniuri)

```go
import "github.com/dchest/uniuri"
```

Package uniuri generates random strings good for use in URIs to identify
unique objects.

Example usage:

```go
s := uniuri.New() // s is now "apHCJBl7L1OmC57n"
```

A standard string created by New() is 16 bytes in length and consists of
Latin upper and lowercase letters, and numbers (from the set of 62 allowed
characters), which means that it has ~95 bits of entropy. To get more
entropy, you can use NewLen(UUIDLen), which returns 20-byte string, giving
~119 bits of entropy, or any other desired length.

Functions read from crypto/rand random source, and panic if they fail to
read from it.


Constants
---------

```go
const (
	// StdLen is a standard length of uniuri string to achive ~95 bits of entropy.
	StdLen = 16
	// UUIDLen is a length of uniuri string to achive ~119 bits of entropy, closest
	// to what can be losslessly converted to UUIDv4 (122 bits).
	UUIDLen = 20
)

```



Variables
---------

```go
var StdChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
```


StdChars is a set of standard characters allowed in uniuri string.


Functions
---------

### func New

```go
func New() string
```

New returns a new random string of the standard length, consisting of
standard characters.

### func NewLen

```go
func NewLen(length int) string
```

NewLen returns a new random string of the provided length, consisting of
standard characters.

### func NewLenChars

```go
func NewLenChars(length int, chars []byte) string
```

NewLenChars returns a new random string of the provided length, consisting
of the provided byte slice of allowed characters (maximum 256).



Public domain dedication
------------------------

Written in 2011-2014 by Dmitry Chestnykh

The author(s) have dedicated all copyright and related and
neighboring rights to this software to the public domain
worldwide. Distributed without any warranty.
http://creativecommons.org/publicdomain/zero/1.0/

