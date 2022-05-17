#!/bin/bash -x

# what i did (for blog post)
# docs: https://faun.pub/understanding-go-mod-and-go-sum-5fd7ec9bcc34
# to read: https://golangbyexample.com/go-mod-sum-module/
# to read: https://www.digitalocean.com/community/tutorials/how-to-use-go-modules
# 
# go mod init (this created go.mod)
# go mod tidy (this downlaoded modules and filled in go.mod and go.sum)

local_workdir=$(cd $(dirname $(dirname "${BASH_SOURCE[0]}")) >/dev/null 2>&1 && pwd)

docker build -f Dockerfile.build -t jforman/waldo:latest .
