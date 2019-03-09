FROM golang:1 as builder
WORKDIR /go/src/github.com/jforman/waldo
COPY waldo.go ./
RUN go get ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o waldo waldo.go

FROM alpine:3.9
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /go/src/github.com/jforman/waldo .
