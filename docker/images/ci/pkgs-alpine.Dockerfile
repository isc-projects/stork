FROM golang:1.26.7-alpine3.23

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.4 \
    ruby-dev~3.4 \
    openjdk17-jre-headless~17 \
    python3~3.12 \
    nodejs~24.17 \
    npm~11.11 \
    protoc~31.1 \
    make~4.4 \
    musl-dev~1.2 \
    mandoc~1.14 \
    gcc~15.2 \
    binutils-gold~2.45
