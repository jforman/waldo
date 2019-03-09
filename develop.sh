#!/bin/bash

docker build -t jforman/waldo:latest .
docker run -it --rm -v `pwd`:/go/src/waldo -v /tmp/creds:/creds -w /go/src/waldo jforman/waldo:latest /bin/bash
