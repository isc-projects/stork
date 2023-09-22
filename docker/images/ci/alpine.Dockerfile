FROM golang:1.21-alpine3.18

RUN apk add --no-cache \
    ruby-rake=13.* \
    ruby-dev=3.* \
    openjdk17-jre-headless=17.* \
    python3=3.11.* \
    nodejs=18.17.* \
    npm=9.* \
    protoc=3.21.* \
    make=4.4.* \
    gcc=12.2.* \
    musl-dev=1.2.* \
    tar=1.34-* \
    binutils-gold=2.40-* \
    mandoc=1.14.*
