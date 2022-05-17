FROM golang:1 as builder
WORKDIR /go/src/github.com/jforman/waldo

COPY waldo.go .
COPY go.mod .
COPY go.sum .
RUN go mod download
