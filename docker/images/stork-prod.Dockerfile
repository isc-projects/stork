# syntax=docker/dockerfile:1.4
# Production-optimized Dockerfile for Stork
# This version focuses on security, performance, and reliability
# while removing optional premium features for simplicity

#################
### Arguments ###
#################

ARG KEA_REPO=public/isc/kea-dev
ARG KEA_VERSION=2.7.6-isc20250128083636
ARG BIND9_VERSION=9.18

###################
### Base images ###
###################

FROM --platform=linux/amd64 debian:12.1-slim AS debian-base
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates=20230311* \
        wget=1.21.* \
        supervisor=4.2.* \
        procps=2:4.0.* \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /var/log/supervisor \
    && chmod 755 /var/log/supervisor

# Build dependencies
FROM debian-base AS builder-base
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        unzip=6.0-* \
        ruby-dev=1:3.* \
        python3=3.11.* \
        python3-venv=3.11.* \
        python3-wheel=0.38.* \
        python3-dev=3.11.* \
        make=4.3-* \
        gcc=4:12.2.* \
        xz-utils=5.4.* \
        libc6-dev=2.36-* \
        openjdk-17-jre-headless=17.0.* \
        git=1:2.39.* \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

#######################
### Build stages    ###
#######################

# Backend dependencies
FROM builder-base AS backend-deps
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Frontend dependencies
FROM builder-base AS frontend-deps
WORKDIR /app/webui
COPY webui/package*.json webui/.npmrc ./
RUN npm ci --only=production

# Build backend with security flags
FROM backend-deps AS backend-builder
COPY . /app/
WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -extldflags=-static" \
    -tags 'netgo osusergo static_build' \
    -o stork-server ./cmd/stork-server \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -installsuffix cgo \
    -ldflags="-w -s -extldflags=-static" \
    -tags 'netgo osusergo static_build' \
    -o stork-agent ./cmd/stork-agent

# Build frontend for production
FROM frontend-deps AS frontend-builder
COPY . /app/
WORKDIR /app/webui
RUN npm run build:prod

#######################
### Final images    ###
#######################

# Production Server image
FROM debian-base AS server
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-client=15+* \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false stork \
    && mkdir -p /var/log/stork /etc/stork \
    && chown -R stork:stork /var/log/stork /etc/stork \
    && chmod -R 755 /var/log/stork \
    && chmod -R 750 /etc/stork

COPY --from=backend-builder /app/backend/stork-server /usr/bin/
COPY --from=frontend-builder /app/webui/dist /usr/share/stork/www
COPY docker/config/supervisor/stork-server.conf /etc/supervisor/conf.d/

USER stork
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/version || exit 1
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]

# Production Agent image
FROM debian-base AS agent
RUN useradd -r -s /bin/false stork \
    && mkdir -p /var/lib/stork-agent /etc/stork \
    && chown -R stork:stork /var/lib/stork-agent /etc/stork \
    && chmod -R 755 /var/lib/stork-agent \
    && chmod -R 750 /etc/stork

COPY --from=backend-builder /app/backend/stork-agent /usr/bin/
COPY docker/config/supervisor/stork-agent.conf /etc/supervisor/conf.d/

USER stork
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/metrics || exit 1
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]

# Production Kea image
FROM debian-base AS kea
ARG KEA_REPO
ARG KEA_VERSION

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl=7.88.* \
        prometheus-node-exporter=1.5.* \
        postgresql-client=15+* \
        apt-transport-https=2.6.* \
        gnupg=2.2.* \
    && wget -qO- https://dl.cloudsmith.io/${KEA_REPO}/cfg/setup/bash.deb.sh | bash \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        isc-kea-ctrl-agent=${KEA_VERSION} \
        isc-kea-dhcp4=${KEA_VERSION} \
        isc-kea-dhcp6=${KEA_VERSION} \
        isc-kea-admin=${KEA_VERSION} \
        isc-kea-common=${KEA_VERSION} \
        isc-kea-hooks=${KEA_VERSION} \
        isc-kea-mysql=${KEA_VERSION} \
        isc-kea-pgsql=${KEA_VERSION} \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false kea \
    && mkdir -p /var/run/kea /etc/kea /etc/stork \
    && chown -R kea:kea /var/run/kea /etc/kea /etc/stork \
    && chmod -R 755 /var/run/kea \
    && chmod -R 750 /etc/kea /etc/stork

COPY docker/config/supervisor/kea-*.conf /etc/supervisor/conf.d/
COPY --from=backend-builder /app/backend/stork-agent /usr/bin/

USER kea
EXPOSE 8080 9547
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD supervisorctl status || exit 1
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]

# Production BIND9 image
FROM internetsystemsconsortium/bind9:${BIND9_VERSION} AS bind9
RUN apk add --no-cache \
        supervisor~=4.2 \
        prometheus-node-exporter~=1.5 \
        procps~=4.0 \
        libc6-compat~=1.2 \
    && rm -rf /var/cache/apk/* \
    && /usr/sbin/rndc-confgen -a \
    && chown bind:bind /etc/bind/* \
    && chmod g+w /etc/bind \
    && mkdir -p /var/log/supervisor /var/lib/stork-agent \
    && chown bind:bind /var/log/supervisor /var/lib/stork-agent \
    && chmod 755 /var/log/supervisor /var/lib/stork-agent

COPY docker/config/supervisor/named.conf /etc/supervisor/conf.d/
COPY --from=backend-builder /app/backend/stork-agent /usr/bin/

USER bind
EXPOSE 53/udp 53/tcp 8080 9119
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD rndc status || exit 1
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
