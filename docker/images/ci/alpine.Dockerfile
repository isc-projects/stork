FROM golang:1.23.1-alpine3.19

RUN apk add --no-cache \
    ruby-rake~13 \
    ruby~3.2 \
    ruby-dev~3.2 \
    openjdk17-jre-headless~17 \
    python3~3.11 \
    nodejs~20 \
    npm~10.2 \
    protoc~24.4 \
    make~4.4 \
    musl-dev~1.2 \
    mandoc~1.14 \
    gcc~13.2

