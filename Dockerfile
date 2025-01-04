FROM golang:1 AS builder
WORKDIR /go/src/github.com/jforman/waldo

COPY waldo.go .
COPY go.mod .
COPY go.sum .
RUN go mod download
