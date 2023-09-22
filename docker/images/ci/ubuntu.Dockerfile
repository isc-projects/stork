FROM ubuntu:22.04

WORKDIR /repo
RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https=2.4.* \
        build-essential=12.* \
        ca-certificates=20230311* \
        curl=7.* \
        git=1:2.34.* \
        gnupg-agent=2.* \
        openjdk-11-jre-headless=11.* \
        python3-dev=3.10.* \
        python3-venv=3.10.* \
        python3-wheel=0.37.* \
        python3-distutils=3.10.* \
        make=4.* \
        man-db=2.* \
        ruby=1:3.0* \
        ruby-dev=1:3.0* \
        postgresql-client=14+* \
        software-properties-common=0.99.* \
        unzip=6.* \
        wget=1.21.* \
        chromium-browser=1:85.* \
        # Clean up cache.
        && rm -rf /var/lib/apt/lists/* \
        # Replace default Python.
        && rm -f /usr/bin/python3 \
        && ln -s /usr/bin/python3.8 /usr/bin/python3 \
        # Ubuntu has dpkg configured to ignore man files by default.
        && rm /etc/dpkg/dpkg.cfg.d/excludes
