FROM ubuntu:18.04

WORKDIR /repo
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https \
        build-essential \
        ca-certificates \
        curl \
        git \
        gnupg-agent \
        openjdk-11-jre-headless \
        python3-sphinx \
        python3-sphinx-rtd-theme \
        ruby \
        ruby-dev \
        rubygems \
        software-properties-common \
        unzip \
        wget
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io
RUN rm -rf /var/lib/apt/lists/*
RUN gem install --no-document --no-rdoc --no-ri fpm
