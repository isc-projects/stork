FROM docker:28

RUN apk add --no-cache \
    openjdk17-jre-headless~17 \
    python3~3.12 \
    openssl~3.5 \
    ruby-rake~13 \
    nodejs~22.16 \
    npm~11 \
    protoc~29.4
