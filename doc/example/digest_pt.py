#!/usr/bin/python -u
#-*- coding: utf-8 -*-

import sys, re, subprocess
import os.path
reload(sys)
sys.setdefaultencoding("utf-8")

SOAR_ARGS=["-ignore-rules=OK"]
USE_DATABASE=""

# 打印pt-query-digest的统计信息
def printStatInfo(buf):
    if buf.strip() == "":
        return
    if re.match("^# Query [0-9]", buf):
        sys.stdout.write(buf.split(":", 1)[0] + "\n")
    sys.stdout.write("\n```text\n")
    sys.stdout.write(buf)
    sys.stdout.write("```\n")

# 打印每条SQL的SOAR结果
def printSqlAdvisor(buf):
    global USE_DATABASE
    buf = re.sub("\\\G$", "", USE_DATABASE + buf)
    if buf.strip() == "":
        return

    cmd = ["soar"]
    if len(SOAR_ARGS) > 0:
        cmd = cmd + SOAR_ARGS

    p = subprocess.Popen(["soar"], stdout=subprocess.PIPE, stdin=subprocess.PIPE)
    adv = p.communicate(input=buf)[0]

    # 清理环境
    USE_DATABASE = ""

    # 删除第一行"# Query: xxxxx"
    try:
        adv = adv.split('\n', 1)[1]
    except:
        pass
    sys.stdout.write(adv + "\n")

# 从统计信息中获取database信息
def getUseDB(line):
    global USE_DATABASE
    USE_DATABASE = "USE " + re.sub(' +', " ", line).split(" ")[2] + ";"

def parsePtQueryDisget(f):
    statBuf = ""
    sqlBuf = ""
    for line in f:
        if line.strip() == "":
            continue

        if line.startswith("#"):
            if line.startswith("# Databases ") and not line.strip().endswith("more"):
                getUseDB(line)
            if re.match("^# Query [0-9]", line):
                # pt-query-digest的头部统计信息
                if line.startswith("# Query 1:"):
                    sys.stdout.write("# pt-query-digest统计信息" + "\n")
                printStatInfo(statBuf)
                statBuf = line
            else:
                statBuf += line
        else:
            if not line.strip().endswith("\G"):
                sqlBuf += line 
            else:
                sqlBuf += line
                printStatInfo(statBuf)
                statBuf = ""
                printSqlAdvisor(sqlBuf)
                sqlBuf = ""

def main():
    global SOAR_ARGS
    if len(sys.argv) == 1:
        f = sys.stdin
        parsePtQueryDisget(f)
    else:
        if os.path.isfile(sys.argv[-1]):
            SOAR_ARGS = sys.argv[1:-1]
            f = open(sys.argv[-1])
        else:
            SOAR_ARGS = sys.argv[1:]
            f = sys.stdin
        parsePtQueryDisget(f)

if __name__ == '__main__':
    main()
