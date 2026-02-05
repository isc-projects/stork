FROM golang:1.25.7-alpine3.22

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.4 \
    ruby-dev~3.4 \
    openjdk17-jre-headless~17 \
    python3~3.12 \
    nodejs~22.22 \
    npm~11.6 \
    protoc~29.4 \
    make~4.4 \
    musl-dev~1.2 \
    mandoc~1.14 \
    gcc~14.2 \
    binutils-gold~2.44
