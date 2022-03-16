FROM ubuntu:18.04

WORKDIR /repo
RUN \
        apt-get update && \
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ruby ruby-dev rubygems build-essential git wget unzip openjdk-11-jre-headless python3-sphinx python3-sphinx-rtd-theme \
        apt-transport-https ca-certificates curl gnupg-agent software-properties-common && \
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
        add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
        apt-get update && \
        DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io && \
        rm -rf /var/lib/apt/lists/* && \
        gem install --no-document --no-ri --no-rdoc fpm
