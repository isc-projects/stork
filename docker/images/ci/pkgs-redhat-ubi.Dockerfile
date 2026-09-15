FROM redhat/ubi10:10.0

# To update the Go version, go to https://go.dev/dl/, find suitable
# version, also get the linux-amd64 and linux-arm64 SHA256 sums.
# In the future, we could semi automate it using https://go.dev/dl/?mode=json
ARG GO_VERSION=1.26.8
ARG GO_SHA256_AMD64=d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b
ARG GO_SHA256_ARM64=211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0

ENV PATH="/root/go/bin:/usr/local/go/bin:${PATH}"

WORKDIR /repo
RUN dnf install -y \
    git-2.52.* \
    java-21-openjdk-headless-21.0.* \
    tzdata-java-2026c \
    man-db-2.12.* \
    make-4.* \
    nodejs-22.23.* \
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
    && ln -s /usr/bin/python3.12 /usr/bin/python3 \
    # Ruby bundler rejects installing packages if the temporary directory is
    # world-writeable.
    && chmod +t /tmp \
    # Git sometimes gets grumpy if the host repo was cloned by a user,
    # with a different UID than the one running the container.
    && git config --global --add safe.directory /app  \
    # Install latest Go. We chose to install version from the upstream to
    # ensure that we always can use the lastest version and not rely on
    # the version that is available in the package manager.
    && ARCH="${TARGETARCH:-$(uname -m)}" \
    && case "${ARCH}" in \
        amd64|x86_64) GO_ARCH=amd64; GO_SHA256="${GO_SHA256_AMD64}" ;; \
        arm64|aarch64) GO_ARCH=arm64; GO_SHA256="${GO_SHA256_ARM64}" ;; \
        *) echo "unsupported architecture: ${ARCH}" >&2; exit 1 ;; \
    esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz \
    && echo "${GO_SHA256}  /tmp/go.tar.gz" | sha256sum -c - \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz
