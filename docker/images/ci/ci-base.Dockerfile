FROM debian:13.5-slim

WORKDIR /repo
RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https=3.0.* \
        build-essential=12.* \
        ca-certificates=20250419* \
        curl=8.* \
        git=1:2.47.* \
        gnupg-agent=2.* \
        openjdk-21-jre-headless=21.* \
        python3=3.13.* \
        python3-pip=25.* \
        python3-dev=3.13.* \
        python3-venv=3.13.* \
        python3-wheel=0.46.* \
        make=4.* \
        man-db=2.* \
        ruby=1:3.3* \
        ruby-dev=1:3.3* \
        postgresql-client=17+* \
        ssh=1:10.* \
        unzip=6.* \
        wget=1.25.* \
        # Chromium for AMD64 architecture is available in 139 version.
        # For ARM64 architecture, it is available in 138 version.
        chromium=15* \
        # Clean up cache.
        && rm -rf /var/lib/apt/lists/* \
        # Replace default Python.
        && rm -f /usr/bin/python3 \
        && ln -s /usr/bin/python3.11 /usr/bin/python3 \
        # Debian has dpkg configured to ignore man files by default.
        && rm /etc/dpkg/dpkg.cfg.d/docker
