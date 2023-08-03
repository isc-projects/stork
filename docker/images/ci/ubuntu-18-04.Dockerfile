FROM ubuntu:18.04

WORKDIR /repo
RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https \
        build-essential \
        ca-certificates \
        curl \
        git \
        gnupg-agent \
        openjdk-11-jre-headless \
        python3.8-dev \
        python3.8-venv \
        make \
        ruby \
        ruby-dev \
        rubygems \
        postgresql-client \
        software-properties-common \
        unzip \
        wget \
        && rm -rf /var/lib/apt/lists/* \
        && rm /usr/bin/python3 \
        && ln -s /usr/bin/python3.8 /usr/bin/python3
