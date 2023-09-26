FROM redhat/ubi8:8.8

WORKDIR /repo
RUN dnf install -y \
    gcc-8.5.* \
    git-2.39.* \
    java-17-openjdk-headless-17.0.* \
    tzdata-java-2023c-* \
    make-4.2.* \
    man-db-2.7.* \
    procps-ng-3.3.* \
    python3.11-3.11.* \
    python3-virtualenv-15.1.* \
    rpm-build-4.14.* \
    rubygem-rake-12.3.* \
    ruby-devel-2.5.* \
    unzip-6.0 \
    wget-1.19.* \
    gcc-c++-8.5.* \
    # Clean up cache.
    && dnf clean all \
    # Replace default Python.
    && rm -f /usr/bin/python3 \
    && ln -s /usr/bin/python3.11 /usr/bin/python3 \
    # Ruby bundler rejects installing packages if the temporary directory is
    # world-writeable.
    && chmod +t /tmp
