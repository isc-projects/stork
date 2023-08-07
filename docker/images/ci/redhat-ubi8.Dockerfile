FROM redhat/ubi8:8.6

WORKDIR /repo
RUN dnf install -y \
    gcc-8.5.* \
    git-2.39.* \
    java-11-openjdk-headless-1:11.0.* \
    tzdata-java-2023c-* \
    make-1:4.2.* \
    man \
    procps-ng-3.3.* \
    python38-3.8.* \
    python3-virtualenv-15.1.* \
    rpm-build-4.14.* \
    rubygem-rake-12.3.* \
    ruby-devel-2.5.* \
    unzip-6.0 \
    wget-1.19.* \
    # Clean up cache.
    && dnf clean all \
    # Replace default Python.
    && rm -f /usr/bin/python3 \
    && ln -s /usr/bin/python3.8 /usr/bin/python3
