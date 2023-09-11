# The main purpose of this container is to run Stork Environment Simulator.
FROM ubuntu:18.04
WORKDIR /sim

# Install essentials.
RUN apt-get update \
    && apt-get install \
            -y \
            --no-install-recommends \
            sudo curl ca-certificates gnupg apt-transport-https \
            supervisor python3-pip python3-setuptools python3-wheel \
            libbind-dev libkrb5-dev libssl-dev libcap-dev libxml2-dev \
            libjson-c-dev libgeoip-dev libprotobuf-c-dev libfstrm-dev \
            liblmdb-dev libssl-dev dnsutils build-essential autoconf \
            autotools-dev automake libtool git cmake libldns-dev \
            libgnutls28-dev \
    # Install libuv for DNS testing.
    && mkdir -p /tmp/libuv \
    && cd /tmp/libuv \
    && git clone https://github.com/libuv/libuv.git \
    && cd libuv \
    && sh autogen.sh \
    && ./configure \
    && make && make install \
    # Install flamethrower for DNS testing.
    && mkdir -p /tmp/flamethrower \
    && cd /tmp/flamethrower \
    && git clone https://github.com/DNS-OARC/flamethrower \
    && cd flamethrower \
    && git checkout v0.10.2 \
    && mkdir build \
    && cd build \
    && cmake .. \
    && make \
    && make install \
    # Install perfdhcp
    && curl -1sLf 'https://dl.cloudsmith.io/public/isc/kea-2-4/cfg/setup/bash.deb.sh' | bash \
    && apt-get update \
    && apt-get install \
        -y \
        --no-install-recommends \
        isc-kea-admin=2.4.0-isc20230630120747 \
        isc-kea-common=2.4.0-isc20230630120747 \
        isc-kea-perfdhcp=2.4.0-isc20230630120747 \
    && mkdir -p /var/run/kea/ \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Setup sim.
COPY tests/sim/requirements.txt /sim
RUN pip3 install --no-cache-dir -r /sim/requirements.txt
COPY tests/sim/index.html tests/sim/sim.py /sim/

# Start flask app.
CMD FLASK_ENV=development FLASK_APP=sim.py LC_ALL=C.UTF-8 LANG=C.UTF-8 flask run --host 0.0.0.0
