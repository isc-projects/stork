FROM debian:12.1-slim

WORKDIR /repo
RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https=2.6.* \
        build-essential=12.* \
        ca-certificates=20230311* \
        curl=7.* \
        git=1:2.39.* \
        gnupg-agent=2.* \
        openjdk-17-jre-headless=17.* \
        python3=3.11.* \
        python3-pip=23.* \
        python3-dev=3.11.* \
        python3-venv=3.11.* \
        python3-wheel=0.38.* \
        python3-distutils=3.11.* \
        make=4.* \
        man-db=2.* \
        ruby=1:3.1* \
        ruby-dev=1:3.1* \
        postgresql-client=15+* \
        software-properties-common=0.99.* \
        ssh=1:9.* \
        unzip=6.* \
        wget=1.21.* \
        chromium=117.* \
        # Clean up cache.
        && rm -rf /var/lib/apt/lists/* \
        # Replace default Python.
        && rm -f /usr/bin/python3 \
        && ln -s /usr/bin/python3.11 /usr/bin/python3 \
        # Debian has dpkg configured to ignore man files by default.
        && rm /etc/dpkg/dpkg.cfg.d/docker
