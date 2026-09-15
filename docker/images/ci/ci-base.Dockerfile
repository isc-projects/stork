FROM debian:13.5-slim

# To update the Go version, go to https://go.dev/dl/, find suitable
# version, also get the linux-amd64 and linux-arm64 SHA256 sums.
# In the future, we could semi automate it using https://go.dev/dl/?mode=json
ARG GO_VERSION=1.26.8
ARG GO_SHA256_AMD64=d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b
ARG GO_SHA256_ARM64=211ffced9dcb9633a55eac6364816ec0ddd951389a740e88fa8b3337971bdda0

ENV PATH="/root/go/bin:/usr/local/go/bin:${PATH}"

WORKDIR /repo
RUN apt-get update \
        && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        apt-transport-https=3.0.* \
        build-essential=12.* \
        ca-certificates=20250419* \
        curl=8.* \
        git=1:2.47.* \
        gnupg-agent=2.* \
        openjdk-21-jre-headless=21.* \
        python3=3.13.* \
        python3-pip=25.* \
        python3-dev=3.13.* \
        python3-venv=3.13.* \
        python3-wheel=0.46.* \
        make=4.* \
        man-db=2.* \
        nodejs=20.* \
        npm=9.* \
        ruby=1:3.3* \
        ruby-dev=1:3.3* \
        postgresql-client=17+* \
        ssh=1:10.* \
        unzip=6.* \
        wget=1.25.* \
        # Chromium is available in 150 version in Debian 13.5.
        chromium=15* \
        # Clean up cache.
        && rm -rf /var/lib/apt/lists/* \
        # Debian has dpkg configured to ignore man files by default.
        && rm /etc/dpkg/dpkg.cfg.d/docker \
        # Git sometimes gets grumpy if the host repo was cloned by a user,
        # with a different UID than the one running the container.
        && git config --global --add safe.directory /app \
        # Install latest Go. We chose to install version from the upstream to
        # ensure that we always can use the lastest version and not rely on
        # the version that is available in Debian.
        && ARCH="${TARGETARCH:-$(dpkg --print-architecture)}" \
        && case "${ARCH}" in \
            amd64) GO_ARCH=amd64; GO_SHA256="${GO_SHA256_AMD64}" ;; \
            arm64) GO_ARCH=arm64; GO_SHA256="${GO_SHA256_ARM64}" ;; \
            *) echo "unsupported architecture: ${ARCH}" >&2; exit 1 ;; \
        esac \
        && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz \
        && echo "${GO_SHA256}  /tmp/go.tar.gz" | sha256sum -c - \
        && rm -rf /usr/local/go \
        && tar -C /usr/local -xzf /tmp/go.tar.gz \
        && rm /tmp/go.tar.gz
