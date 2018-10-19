#!/bin/bash

## Go version check
GO_VERSION_MIN=$1
echo "==> Checking that build is using go version >= ${GO_VERSION_MIN}..."

GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+\(\.[0-9]\+\)\?' | tr -d 'go')

IFS="." read -r -a GO_VERSION_ARR <<<"$GO_VERSION"
IFS="." read -r -a GO_VERSION_REQ <<<"$GO_VERSION_MIN"

if [[ ${GO_VERSION_ARR[0]} -lt ${GO_VERSION_REQ[0]} || (${GO_VERSION_ARR[0]} -eq ${GO_VERSION_REQ[0]} && (${GO_VERSION_ARR[1]} -lt ${GO_VERSION_REQ[1]} || (${GO_VERSION_ARR[1]} -eq ${GO_VERSION_REQ[1]} && ${GO_VERSION_ARR[2]} -lt ${GO_VERSION_REQ[2]}))) ]] \
    ; then
    echo "requires go $GO_VERSION_MIN to build; found $GO_VERSION."
    exit 1
fi

## Generate Repository Version
version=$(git log --date=iso --pretty=format:"%cd @%h" -1)
if [ "X${version}" == "X" ]; then
    version="not a git repo"
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
