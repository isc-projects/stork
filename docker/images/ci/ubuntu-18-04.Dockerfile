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
        python3.8-distutils \
        make \
        man \
        ruby \
        ruby-dev \
        rubygems \
        postgresql-client \
        software-properties-common \
        unzip \
        wget \
        chromium-browser \
        # Clean up cache.
        && rm -rf /var/lib/apt/lists/* \
        # Replace default Python.
        && rm -f /usr/bin/python3 \
        && ln -s /usr/bin/python3.8 /usr/bin/python3 \
        # Ubuntu has dpkg configured to ignore man files by default.
        && rm /etc/dpkg/dpkg.cfg.d/excludes
