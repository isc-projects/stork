FROM golang:1.21-alpine3.17

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.1 \
    ruby-dev~3.1 \
    openjdk17-jre-headless~17 \
    python3~3.10 \
    nodejs~18.17 \
    npm~9 \
    protoc~3.21 \
    make~4.3 \
    gcc~12.2 \
    musl-dev~1.2 \
    tar~1.34 \
    binutils-gold~2.39 \
    mandoc~1.14
