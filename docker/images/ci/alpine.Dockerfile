FROM golang:1.22-alpine3.18

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.2 \
    ruby-dev~3.2 \
    openjdk17-jre-headless~17 \
    python3~3.11 \
    nodejs~18 \
    npm~9.6 \
    protoc~3 \
    make~4.4 \
    gcc~12.2 \
    musl-dev~1.2 \
    tar~1.34 \
    binutils-gold~2.40 \
    mandoc~1.14 \
    g++~12.2
