FROM golang:1.23.5-alpine3.20

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.3 \
    ruby-dev~3.3 \
    openjdk17-jre-headless~17 \
    python3~3.12 \
    nodejs~20 \
    npm~10.9 \
    protoc~24.4 \
    make~4.4 \
    musl-dev~1.2 \
    mandoc~1.14 \
    gcc~13.2 \
    binutils-gold~2.42
