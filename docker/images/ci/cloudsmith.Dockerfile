FROM ubuntu:18.04

RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        locales \
        python3-pip \
        python3-setuptools \
        python3-wheel \
        rake \
        wget
RUN rm -rf /var/lib/apt/lists/*
RUN locale-gen en_US.UTF-8
RUN update-locale LANG='en_US.UTF-8' LC_ALL='en_US.UTF-8'
RUN LANG='en_US.UTF-8' LC_ALL='en_US.UTF-8' pip3 install --upgrade cloudsmith-cli
