#!/bin/bash

## Generate Repository Version
tag="$(git describe --tags --always)"
version="$(git log --date=iso --pretty=format:"%cd" -1) ${tag}"
if [ "X${version}" == "X" ]; then
    version="not a git repo"
    tag="not a git repo"
fi

git_dirty=$(git diff --no-ext-diff 2>/dev/null | wc -l)

compile="$(date +"%F %T %z") by $(go version)"

branch=$(git rev-parse --abbrev-ref HEAD)

dev_path=$(
    cd "$(dirname "$0")" || exit
    pwd
)

cat <<EOF | gofmt >common/version.go
package common

// -version输出信息
const (
    Version = "${version}"
    Compile = "${compile}"
    Branch  = "${branch}"
    GitDirty= ${git_dirty}
    DevPath = "${dev_path}"
)
EOF

XIAOMI=$(git ls-remote --get-url | grep XiaoMi)
if [ "x${XIAOMI}" != "x" ]; then
    echo "${tag}" | awk -F '-' '{print $1}' > VERSION
fi
