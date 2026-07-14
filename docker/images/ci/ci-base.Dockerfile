FROM debian:13.5-slim

# Go release metadata: https://go.dev/dl/?mode=json
ARG GO_VERSION=1.26.5
ARG GO_SHA256_AMD64=5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053
ARG GO_SHA256_ARM64=fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49

ENV PATH="/usr/local/go/bin:${PATH}"

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
        && rm /etc/dpkg/dpkg.cfg.d/docker

ARG GO_VERSION
ARG GO_SHA256_AMD64
ARG GO_SHA256_ARM64
RUN ARCH="${TARGETARCH:-$(dpkg --print-architecture)}" \
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
