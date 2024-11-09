# Build container
FROM golang:1.22 AS builder

RUN go version

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates git zlib1g-dev

COPY . /go/src/github.com/TicketsBot/misconduct-detector
WORKDIR /go/src/github.com/TicketsBot/misconduct-detector

RUN git submodule update --init --recursive --remote

RUN set -Eeux && \
    go mod download && \
    go mod verify

RUN GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -o main cmd/misconduct-detector/main.go

# Prod container
FROM ubuntu:latest

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates curl

COPY --from=builder /go/src/github.com/TicketsBot/misconduct-detector/main /srv/misconduct-detector/main

RUN chmod +x /srv/misconduct-detector/main

RUN useradd -m container
USER container
WORKDIR /srv/misconduct-detector

CMD ["/srv/misconduct-detector/main"]