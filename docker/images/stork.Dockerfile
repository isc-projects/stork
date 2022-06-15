#################
### Arguments ###
#################

ARG KEA_REPO=public/isc/kea-2-0
ARG KEA_VER=2.0.2-isc20220227221539
# Indicate if the premium packages should be installed.
# Valid values: "premium" or empty.
ARG KEA_PREMIUM=""

###################
### Base images ###
###################

FROM debian:11.2-slim AS debian-base
RUN apt-get update \
        # System-wise dependencies
        && apt-get install \
                -y \
                --no-install-recommends \
                ca-certificates=20210119 \
                wget=1.21-* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
ENV CI=true

# Install system-wide dependencies
FROM debian-base AS base
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
        # System-wise dependencies
        && apt-get install \
                -y \
                --no-install-recommends \
                unzip=6.0-* \
                ruby-dev=1:2.7+* \
                python3.9=3.9.* \
                python3-venv=3.9.* \
                make=4.3-* \
                gcc=4:10.2.* \
                xz-utils=5.2.* \
                libc6-dev=2.31-* \
                ruby-rubygems=3.2.* \
                openjdk-11-jre-headless=11.0.* \
                git=1:2.30.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

#############
### Stork ###
#############

# Install main dependencies
FROM base AS prepare 
WORKDIR /app/rakelib
COPY rakelib/00_init.rake ./
WORKDIR /app/rakelib/init_deps
COPY rakelib/init_deps ./
WORKDIR /app
COPY Rakefile ./
# It must be split into separate stages.
RUN rake prepare

# Backend dependencies installation
FROM prepare AS gopath-prepare
WORKDIR /app/rakelib
COPY rakelib/10_codebase.rake ./
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN rake prepare:backend_deps

# Frontend dependencies installation
FROM prepare AS nodemodules-prepare
WORKDIR /app/rakelib
COPY rakelib/10_codebase.rake ./
WORKDIR /app/webui
COPY webui/package.json webui/package-lock.json ./
RUN rake prepare:ui_deps

# General-purpose stage for tasks: building, testing, linting, etc.
# It contains the codebase with dependencies
FROM prepare AS codebase
WORKDIR /app/tools/golang
COPY --from=gopath-prepare /app/tools/golang .
WORKDIR /app/webui
COPY --from=nodemodules-prepare /app/webui .
WORKDIR /app
COPY Rakefile .
WORKDIR /app/api
COPY api .
WORKDIR /app/backend
COPY backend .
WORKDIR /app/doc
COPY doc .
WORKDIR /app/etc
COPY etc .
WORKDIR /app/grafana
COPY grafana .
WORKDIR /app/rakelib
COPY rakelib/10_codebase.rake rakelib/20_build.rake rakelib/40_dist.rake ./
WORKDIR /app/webui
COPY webui .

# Build the Stork binaries
FROM codebase AS server-builder
RUN rake build:server_only_dist

FROM codebase AS webui-builder
RUN rake build:ui_only_dist

FROM codebase AS agent-builder
RUN rake build:agent_dist

# Agent container
FROM debian-base as agent
COPY --from=agent-builder /app/dist/agent /
ENTRYPOINT [ "/usr/bin/stork-agent" ]
# Incoming port
EXPOSE 8080
# Prometheus Kea port
EXPOSE 9547
# Prometheus Bing9 port
EXPOSE 9119

# Server container
FROM debian-base AS server
COPY --from=server-builder /app/dist/server/ /
ENTRYPOINT [ "/usr/bin/stork-server" ]
EXPOSE 8080
HEALTHCHECK CMD [ "wget", "--delete-after", "-q", "http://localhost:8080/api/version" ]

# Web UI container
FROM nginx:1.21-alpine AS webui
ENV CI=true
COPY --from=webui-builder /app/dist/server/ /
COPY webui/nginx.conf /tmp/nginx.conf.tpl
ENV DOLLAR=$
ENV API_HOST localhost
ENV API_PORT 5000
ENTRYPOINT [ "/bin/sh", "-c", \
        "envsubst < /tmp/nginx.conf.tpl > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'" ]
EXPOSE 80
HEALTHCHECK CMD ["curl", "--fail", "http://localhost:80"]

# Server with webui container
FROM debian-base AS server-webui
COPY --from=server-builder /app/dist/server /
COPY --from=webui-builder /app/dist/server /
ENTRYPOINT [ "/usr/bin/stork-server" ]
EXPOSE 8080
HEALTHCHECK CMD [ "wget", "--delete-after", "-q", "http://localhost:8080/api/version" ]

#################################
### Kea / Bind9 + Stork Agent ###
#################################

# Kea config generator
FROM base AS kea-config-generator
RUN mkdir -p /etc/kea && touch /etc/kea/kea-dhcp4.conf
WORKDIR /app/docker/tools
COPY docker/tools/gen-kea-config.py .
ENTRYPOINT [ "python3", "/app/docker/tools/gen-kea-config.py", "-o", "/etc/kea/kea-dhcp4.conf" ]
CMD [ "7000" ]

# Kea with Stork Agent container
FROM debian-base AS kea-base
# Install Kea dependencies
RUN apt-get update \
        && apt-get install \
                -y \
                --no-install-recommends \
                curl=7.74.* \
                supervisor=4.2.* \
                prometheus-node-exporter=1.1.* \
                default-mysql-client=1.0.* \ 
                postgresql-client=13+* \
                apt-transport-https=2.2.* \
                gnupg=2.2.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
# Install Kea from Cloudsmith
SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]
ARG KEA_REPO
ARG KEA_VER
RUN wget -q -O- https://dl.cloudsmith.io/${KEA_REPO}/cfg/setup/bash.deb.sh | bash \
        && apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                isc-kea-ctrl-agent=${KEA_VER} \
                isc-kea-dhcp4-server=${KEA_VER} \
                isc-kea-dhcp6-server=${KEA_VER} \
                isc-kea-admin=${KEA_VER} \
                isc-kea-common=${KEA_VER} \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        && mkdir -p /var/run/kea/ \
        # Puts empty credentials file to allow mount it as volume.
        && mkdir -p /etc/stork/ \
        && echo "{}" > /etc/stork/agent-credentials.json

# Install premium packages. The KEA_REPO variable must
# be set to the private repository and include an access token.
# Docker ignores this section if the KEA_PREMIUM is empty - thanks
# to this, the image builds correctly when the token is unknown.
FROM kea-base AS keapremium-base
ARG KEA_PREMIUM
ARG KEA_VER
# Execute only if the premium is enabled
RUN [ "${KEA_PREMIUM}" != "premium" ] || ( \
        apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                isc-kea-premium-host-cmds=${KEA_VER} \
                isc-kea-premium-forensic-log=${KEA_VER} \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        && mkdir -p /var/run/kea/ \
)

# Use the "kea-base" or "keapremium-base" image as a base image
# for this stage.
# hadolint ignore=DL3006
FROM kea${KEA_PREMIUM}-base AS kea
# Install agent    
COPY --from=agent-builder /app/dist/agent /
# Database
WORKDIR /var/lib/db
COPY docker/init/init_db.sh docker/init/init_*_db.sh docker/init/init_*_query.sql ./
# Run
WORKDIR /root
ENV DB_TYPE=mysql
ENV DB_HOST=172.20.0.115
ENV DB_USER=kea
ENV DB_PASSWORD=kea
ENV DB_NAME=kea
ENTRYPOINT [ "/bin/bash", "-c", \
        "/var/lib/db/init_db.sh && supervisord -c /etc/supervisor/supervisord.conf" ]
# Incoming port
EXPOSE 8080
# Prometheus Kea port
EXPOSE 9547
HEALTHCHECK CMD [ "supervisorctl", "status " ]
# Configuration files:
# Mysql database seed: /var/lib/db/init_mysql_query.sql
# Postgres database seed: /var/lib/db/init_pgsql_query.sql
# Supervisor: /etc/supervisor/supervisord.conf
# Kea DHCPv4: /etc/kea/kea-dhcp4.conf
# Kea DHCPv6: /etc/kea/kea-dhcp6.conf
# Kea Control Agent: /etc/kea/kea-ctrl-agent.conf
# Stork Agent files: /etc/stork

# Bind9 with Stork Agent container
FROM debian-base AS bind
# Install Bind9 dependencies
RUN apt-get update \
        && apt-get install \
                -y \
                --no-install-recommends \
                curl=7.74.* \
                supervisor=4.2.* \
                prometheus-node-exporter=1.1.* \
                apt-transport-https=2.2.* \
                gnupg=2.2.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
# Install Bind
ARG BIND_VER="1:9.16.*"
RUN apt-get update \
        && apt-get install \
                -y \
                --no-install-recommends \
                bind9=${BIND_VER} \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        && chown root:bind /etc/bind/rndc.key \
        && chmod 640 /etc/bind/rndc.key \
        # Puts empty database file to allow mounting it as a volume.
        && touch /etc/bind/db.test
# Install agent    
COPY --from=agent-builder /app/dist/agent /
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
# Incoming port
EXPOSE 8080
# Prometheus Bing9 port
EXPOSE 9119
HEALTHCHECK CMD [ "supervisorctl", "status " ]
# Configuration files:
# Supervisor: /etc/supervisor/supervisord.conf
# Stork Agent: /etc/stork
# Bind9 config: /etc/bind/named.conf
# Bind9 database: /etc/bind/db.test
