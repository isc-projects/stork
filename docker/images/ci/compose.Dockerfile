FROM docker:24

RUN apk add --no-cache \
    openjdk17-jre-headless=17.* \
    python3=3.11.* \
    openssl=3.1.* \
    ruby-rake=13.* \
    nodejs=18.17.* \
    npm=9.*
