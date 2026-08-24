FROM ruby:3.3.5-bullseye

# It is a minimal image based on Debian with old GLIBC version that is
# suitable for building Stork Go binaries (and nothing else). It is used to
# build Stork Go binaries that are compatible with older Debian-like
# distributions.

# To update the Go version, go to https://go.dev/dl/, find suitable
# version, also get the linux-amd64 and linux-arm64 SHA256 sums.
# In the future, we could semi automate it using https://go.dev/dl/?mode=json
ARG GO_VERSION=1.26.7
ARG GO_SHA256_AMD64=ffb5f8de10c62550dfddab66b36b57030721e0a44a3218e9e1181d7b59f121ca
ARG GO_SHA256_ARM64=5a4ec883379d51ee9ce1040d5e87f8d35e20387574dd8c947feb01eabc3c1b37

ENV PATH="/root/go/bin:/usr/local/go/bin:${PATH}"

WORKDIR /repo
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
        && rm /tmp/go.tar.gz \
        # Git sometimes gets grumpy if the host repo was cloned by a user,
        # with a different UID than the one running the container.
        && git config --global --add safe.directory /app
