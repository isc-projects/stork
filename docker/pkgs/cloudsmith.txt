FROM ubuntu:18.04

RUN apt-get update && \
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends locales python3-pip python3-setuptools python3-wheel && \
        rm -rf /var/lib/apt/lists/* && \
        locale-gen en_US.UTF-8 && \
        update-locale LANG='en_US.UTF-8' LANGUAGE='en_US:en' LC_ALL='en_US.UTF-8'
RUN LANG='en_US.UTF-8' LANGUAGE='en_US:en' LC_ALL='en_US.UTF-8' pip3 install --upgrade cloudsmith-cli
