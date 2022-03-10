# syntax = docker/dockerfile:1.0-experimental
FROM golang:1.16.10-buster as builder

WORKDIR /app

# download depencites
COPY go.mod .
COPY go.sum .

RUN go mod download

# build project
COPY . .

RUN GOOS=linux go build -a -o ./app ./main.go

FROM debian:buster-slim

RUN set -xe \
    && export DEBIAN_FRONTEND=noninteractive \
    && apt-get update -qq \
    && apt-get dist-upgrade -qq \
    && apt-get install -qq \
        ca-certificates \
    && SUDO_FORCE_REMOVE=yes apt-get autoremove -qq \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /app

COPY --from=builder /app/app .
COPY config.yaml .

ENTRYPOINT ["/app/app"]

CMD ["-v"]
