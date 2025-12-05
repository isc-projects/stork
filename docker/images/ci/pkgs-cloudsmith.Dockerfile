FROM ubuntu:22.04

RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        locales=2.35-* \
        python3-pip=22.* \
        python3-setuptools=59.* \
        python3-wheel=0.37.* \
        rake=13.* \
        wget=1.21.* \
        && rm -rf /var/lib/apt/lists/* \
        && locale-gen en_US.UTF-8 \
        && update-locale LANG='en_US.UTF-8' LC_ALL='en_US.UTF-8' \
        && LANG='en_US.UTF-8' LC_ALL='en_US.UTF-8' pip3 install --no-cache-dir cloudsmith-cli==1.1.1
