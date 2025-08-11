FROM redhat/ubi10:10.0

WORKDIR /repo
RUN dnf install -y \
    git-2.47.* \
    java-21-openjdk-headless-21.0.* \
    tzdata-java-2025b \
    man-db-2.12.* \
    gcc-c++-14.2.* \
    make-4.* \
    procps-ng-4.0.* \
    python3-3.12.* \
    rubygem-rake-13.1.* \
    ruby-devel-3.3.* \
    unzip-6.0 \
    wget-1.24.* \
    xz-5.6.* \
    # Clean up cache.
    && dnf clean all \
    # Replace default Python.
    && rm -f /usr/bin/python3 \
    && ln -s /usr/bin/python3.11 /usr/bin/python3 \
    # Ruby bundler rejects installing packages if the temporary directory is
    # world-writeable.
    && chmod +t /tmp
