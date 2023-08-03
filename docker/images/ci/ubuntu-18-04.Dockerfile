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
        python3-dev \
        python3-venv \
        python3-wheel \
        make \
        ruby \
        ruby-dev \
        rubygems \
        software-properties-common \
        unzip \
        wget \
        && rm -rf /var/lib/apt/lists/*
