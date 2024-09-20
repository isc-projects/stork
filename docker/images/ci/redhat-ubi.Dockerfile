FROM redhat/ubi9:9.4

WORKDIR /repo
RUN dnf install -y \
    git-2.43.* \
    java-17-openjdk-headless-17.0.* \
    tzdata-java-2024a \
    man-db-2.9.* \
    gcc-c++-11.4.* \
    make-4.* \
    procps-ng-3.3.* \
    python3.11-3.11.* \
    rubygem-rake-13.0.* \
    ruby-devel-3.0.* \
    unzip-6.0 \
    wget-1.21.* \
    xz-5.2.* \
    # Clean up cache.
    && dnf clean all \
    # Replace default Python.
    && rm -f /usr/bin/python3 \
    && ln -s /usr/bin/python3.11 /usr/bin/python3 \
    # Ruby bundler rejects installing packages if the temporary directory is
    # world-writeable.
    && chmod +t /tmp
