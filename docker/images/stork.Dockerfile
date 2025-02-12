#################
### Arguments ###
#################

ARG KEA_REPO=public/isc/kea-dev
ARG KEA_VERSION=2.7.0-isc20240624090938
# Indicates if the premium packages should be installed.
# Valid values: "premium" or empty.
ARG KEA_PREMIUM=""
# Indicates what Kea packages should be installed.
ARG KEA_PRIOR_2_3_0="false"
ARG KEA_PRIOR_2_7_5="true"
ARG BIND9_VERSION=9.18

###################
### Base images ###
###################

# The demo setup is not fully compatible with arm64 architectures.
# In particular, only the amd64 image with named is available.
# In addition, the flamethrower program requires an older Debian
# version for which we provide no arm64 packages with perfdhcp.
# Since we use common containers for building Stork, building
# BIND9, Kea and simulator on different architectures is impossible.
# The good news is that amd64 can be emulated on top of the arm64.
FROM --platform=linux/amd64 debian:12.1-slim AS debian-base
RUN apt-get update \
        # System-wise dependencies
        && apt-get install \
                -y \
                --no-install-recommends \
                ca-certificates=20230311* \
                wget=1.21.* \
                supervisor=4.2.* \
                procps=2:4.0.* \
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
COPY webui/package.json webui/package-lock.json webui/.npmrc ./
RUN rake prepare:ui_deps

# General-purpose stage for tasks: building, testing, linting, etc.
# It contains the codebase with dependencies
FROM prepare AS codebase
WORKDIR /app
COPY Rakefile .
WORKDIR /app/doc
COPY doc .
WORKDIR /app/etc
COPY etc .
WORKDIR /app/api
COPY api .
WORKDIR /app/codegen
COPY codegen .
WORKDIR /app/rakelib
COPY rakelib/10_codebase.rake rakelib/20_build.rake rakelib/30_dev.rake rakelib/40_dist.rake ./

FROM codebase AS codebase-backend
WORKDIR /app/tools/golang
COPY --from=gopath-prepare /app/tools/golang .
WORKDIR /app/backend
COPY backend .

FROM codebase-backend AS codebase-hooks
WORKDIR /app/hooks
COPY hooks .
# Remove the empty (not cloned) hook directories.
RUN find . -type d -empty -delete
WORKDIR /app/backend

FROM codebase AS codebase-webui
WORKDIR /app/grafana
COPY grafana .
WORKDIR /app/tools/golang/go
COPY --from=gopath-prepare /app/tools/golang/go .
WORKDIR /app/backend
COPY backend/go.mod ./
COPY backend/go.sum ./
COPY backend/version.go ./
COPY backend/codegen ./codegen
COPY backend/cmd/stork-code-gen ./cmd/stork-code-gen
WORKDIR /app/webui
COPY --from=nodemodules-prepare /app/webui .
WORKDIR /app/webui
COPY webui .

# Build the Stork hooks
FROM codebase-hooks AS hooks-builder
WORKDIR /app/rakelib
COPY rakelib/90_hooks.rake .
WORKDIR /app/backend
RUN rake hook:build

# Build the Stork binaries
FROM codebase-backend AS server-builder
# The lib directory prevents from copying the files in the next stages
# because in Debian Bookworm the /lib directory is a link.
RUN rake build:server_only_dist && rm -rf /app/dist/server/lib

FROM codebase-webui AS webui-builder
# The lib directory prevents from copying the files in the next stages
# because in Debian Bookworm the /lib directory is a link.
RUN rake build:ui_only_dist && rm -rf /app/dist/server/lib

FROM codebase-backend AS agent-builder
# The lib directory prevents from copying the files in the next stages
# because in Debian Bookworm the /lib directory is a link.
RUN rake build:agent_dist && rm -rf /app/dist/agent/lib

FROM codebase AS server-full-builder
COPY --from=server-builder /app/ /app/
COPY --from=webui-builder /app/ /app/
# The lib directory prevents from copying the files in the next stages
# because in Debian Bookworm the /lib directory is a link.
RUN rake build:server_dist && rm -rf /app/dist/server/lib

# Agent container
FROM debian-base AS agent
COPY --from=agent-builder /app/dist/agent /
ENTRYPOINT [ "/usr/bin/stork-agent" ]
# Incoming port
EXPOSE 8080
# Prometheus Kea port
EXPOSE 9547
# Prometheus Bing9 port
EXPOSE 9119

# Server containers
FROM debian-base AS server
COPY --from=server-builder /app/dist/server/ /
COPY --from=hooks-builder /app/hooks/stork-server-*.so /usr/lib/stork-server/hooks/
ENTRYPOINT [ "/bin/sh", "-c", \
        "supervisord -c /etc/supervisor/supervisord.conf" ]
EXPOSE 8080
HEALTHCHECK CMD [ "wget", "--delete-after", "-q", "http://localhost:8080/api/version" ]

# Web UI container
FROM nginx:1.25-alpine AS webui
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

# Web UI container on Apache
FROM httpd:2.4 AS webui-apache
COPY --from=webui-builder /app/dist/server/ /
COPY etc/httpd-stork.conf /usr/local/apache2/conf/httpd.conf
ENV API_HOST localhost
ENV API_PORT 8080
EXPOSE 81
EXPOSE 82
RUN apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                curl=7.88.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
RUN sed -i 's/<base href="\/">/<base href="\/stork\/">/g' /usr/share/stork/www/index.html
HEALTHCHECK CMD ["curl", "--fail", "http://localhost:81"]

#################################
### Kea / Bind9 + Stork Agent ###
#################################

# Kea config generator
FROM base AS kea-config-generator
RUN mkdir -p /etc/kea && touch /etc/kea/kea-dhcp4.conf
WORKDIR /app/docker/tools
COPY docker/tools/gen_kea_config.py .
ENTRYPOINT [ "python3", "/app/docker/tools/gen_kea_config.py", "-o", "/etc/kea/kea-dhcp4.conf" ]
CMD [ "7000" ]

# Kea with Stork Agent container
FROM debian-base AS kea-base
# Install Kea dependencies
RUN apt-get update \
        && apt-get install \
                -y \
                --no-install-recommends \
                curl=7.88.* \
                prometheus-node-exporter=1.5.* \
                default-mysql-client=1.1.* \
                postgresql-client=15+* \
                apt-transport-https=2.6.* \
                gnupg=2.2.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
# Install Kea from Cloudsmith
SHELL [ "/bin/bash", "-o", "pipefail", "-c" ]
ARG KEA_REPO
ARG KEA_VERSION
ARG KEA_PRIOR_2_3_0
ARG KEA_PRIOR_2_7_5
RUN wget --no-verbose -O- https://dl.cloudsmith.io/${KEA_REPO}/cfg/setup/bash.deb.sh | bash \
        && apt-get update \
        && if [ ${KEA_PRIOR_2_3_0} == "true" ]; then \
                apt-get install \
                        --no-install-recommends \
                        -y \
                        python3-isc-kea-connector=${KEA_VERSION} \
                        isc-kea-ctrl-agent=${KEA_VERSION} \
                        isc-kea-dhcp4-server=${KEA_VERSION} \
                        isc-kea-dhcp6-server=${KEA_VERSION} \
                        isc-kea-admin=${KEA_VERSION} \
                        isc-kea-common=${KEA_VERSION} \
                        ;\
           elif [ ${KEA_PRIOR_2_7_5} == "true" ]; then \
                apt-get install \
                        --no-install-recommends \
                        -y \
                        isc-kea-ctrl-agent=${KEA_VERSION} \
                        isc-kea-dhcp4=${KEA_VERSION} \
                        isc-kea-dhcp6=${KEA_VERSION} \
                        isc-kea-admin=${KEA_VERSION} \
                        isc-kea-common=${KEA_VERSION} \
                        isc-kea-hooks=${KEA_VERSION} \
                        isc-kea-perfdhcp=${KEA_VERSION} \
                        ;\
           else \
                apt-get install \
                        --no-install-recommends \
                        -y \
                        isc-kea-ctrl-agent=${KEA_VERSION} \
                        isc-kea-dhcp4=${KEA_VERSION} \
                        isc-kea-dhcp6=${KEA_VERSION} \
                        isc-kea-admin=${KEA_VERSION} \
                        isc-kea-common=${KEA_VERSION} \
                        isc-kea-hooks=${KEA_VERSION} \
                        isc-kea-perfdhcp=${KEA_VERSION} \
                        isc-kea-mysql=${KEA_VERSION} \
                        isc-kea-pgsql=${KEA_VERSION} \
                        ;\
        fi \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        && mkdir -p /var/run/kea/ \
        # Puts empty credentials file to allow mount it as volume.
        && mkdir -p /etc/stork/

# Install premium packages. The KEA_REPO variable must
# be set to the private repository and include an access token.
# Docker ignores this section if the KEA_PREMIUM is empty - thanks
# to this, the image builds correctly when the token is unknown.
FROM kea-base AS keapremium-base
ARG KEA_PREMIUM
ARG KEA_VERSION
# Execute only if the premium is enabled
RUN [ "${KEA_PREMIUM}" != "premium" ] || ( \
        apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                isc-kea-premium-host-cmds=${KEA_VERSION} \
                isc-kea-premium-subnet-cmds=${KEA_VERSION} \
                isc-kea-premium-forensic-log=${KEA_VERSION} \
                isc-kea-premium-host-cache=${KEA_VERSION} \
                isc-kea-premium-radius=${KEA_VERSION} \
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
ENV DB_HOST=172.24.0.115
ENV DB_USER=kea
ENV DB_PASSWORD=kea
ENV DB_NAME=kea
ENTRYPOINT [ "/bin/sh", "-c", \
        "/var/lib/db/init_db.sh && supervisord -c /etc/supervisor/supervisord.conf" ]
# Incoming port
EXPOSE 8080
# Prometheus Kea port
EXPOSE 9547
HEALTHCHECK CMD [ "supervisorctl", "status" ]
# Configuration files:
# Mysql database seed: /var/lib/db/init_mysql_query.sql
# Postgres database seed: /var/lib/db/init_pgsql_query.sql
# Supervisor: /etc/supervisor/supervisord.conf
# Kea DHCPv4: /etc/kea/kea-dhcp4.conf
# Kea DHCPv6: /etc/kea/kea-dhcp6.conf
# Kea Control Agent: /etc/kea/kea-ctrl-agent.conf
# Stork Agent files: /etc/stork

# Bind9 with Stork Agent container
FROM internetsystemsconsortium/bind9:${BIND9_VERSION} AS bind
# Install Bind9 dependencies
# BIND9 images switched to Alpine 3.20 some time around Aug 2024.
# The --no-cache prevents any local caching (storing anything
# in /var/cache/apk). We use trust-but-verify, so later we delete
# the dir anyway.
#
# Once we have native musl builds, we should remove the libc6-compat
# package.
RUN apk update \
        && apk add \
        --no-cache \
        --no-interactive \
        supervisor \
        prometheus-node-exporter \
        procps \
        libc6-compat \
        && rm -rf /var/cache/apk/* \
        # Alpine specific: need to generate rndc key on our own.
        && /usr/sbin/rndc-confgen -a \
        && chown bind:bind /etc/bind/* \
        && chmod g+w /etc/bind \
        # Puts empty database file to allow mounting it as a volume.
        && touch /etc/bind/db.test \
        # The bind image uses a dedicated user. We need to run the entry point
        # with this user to allow Stork Agent to read the process info. It
        # means that the supervisors will use the same user. It must have the
        # right to write the log files.
        && mkdir -p /var/log/supervisor \
        && chown bind:bind /var/log/supervisor \
        && chmod 755 /var/log/supervisor \
        # The same situation is with the Stork Agent
        && mkdir -p /var/lib/stork-agent \
        && chown bind:bind /var/lib/stork-agent \
        && chmod 755 /var/lib/stork-agent
# Install agent
COPY --from=agent-builder /app/dist/agent/usr/bin /usr/bin
# Use dedicated bind user
USER bind
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
# Incoming port
EXPOSE 8080
# Prometheus Bind9 port
EXPOSE 9119
HEALTHCHECK CMD [ "supervisorctl", "-c", "/etc/supervisor/supervisord.conf", "status" ]
# Configuration files:
# Supervisor: /etc/supervisor/supervisord.conf
# Stork Agent: /etc/stork
# Bind9 config: /etc/bind/named.conf
# Bind9 database: /etc/bind/db.test

FROM bind AS bind-chroot
# Chroot must be called by the root user.
USER root
# Prepare a directory for the chroot environment.
RUN mkdir -p /chroot/etc \
        # Copy the configuration files.
        && cp -R -p /etc/bind /chroot/etc/bind \
        && rm -rf /etc/bind \
        # Create the necessary directories.
        && mkdir -p /chroot/var/cache/bind \
        && mkdir -p /chroot/run/named \
        && chown -R bind:bind /chroot

# PowerDNS.
FROM debian-base AS pdns
RUN apt-get update \
        && apt-get install \
                -y \
                --no-install-recommends \
                prometheus-node-exporter=1.5.* \
                pdns-server=4.7.* \
                pdns-backend-bind=4.7.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        && mkdir -p /var/log/supervisor \
        && mkdir -p /var/lib/stork-agent
# Install agent
COPY --from=agent-builder /app/dist/agent /
# Run the processes.
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
HEALTHCHECK CMD [ "supervisorctl", "-c", "/etc/supervisor/supervisord.conf", "status" ]

#################
### Packaging ###
#################

FROM server-full-builder AS server_package_builder
RUN rake build:server_pkg && rake utils:remove_last_package_suffix

FROM agent-builder AS agent_package_builder
RUN rake build:agent_pkg && rake utils:remove_last_package_suffix

FROM debian-base AS external-packages
RUN apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                curl=7.88.* \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/* \
        # The post-install hooks of the packages call the systemctl command
        # and fail if it doesn't exist. The empty script is a simple
        # workaround for this missing command.
        && touch /usr/bin/systemctl \
        && chmod a+x /usr/bin/systemctl
ARG STORK_CS_VERSION=*
RUN wget --no-verbose -O- https://dl.cloudsmith.io/public/isc/stork/cfg/setup/bash.deb.sh | bash \
        && apt-get update \
        && apt-get install \
                --no-install-recommends \
                -y \
                isc-stork-agent=${STORK_CS_VERSION} \
                isc-stork-server=${STORK_CS_VERSION} \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*
COPY --from=server_package_builder /app/dist/pkgs/isc-stork-server.deb /app/dist/pkgs/isc-stork-server.deb
COPY --from=agent_package_builder /app/dist/pkgs/isc-stork-agent.deb /app/dist/pkgs/isc-stork-agent.deb
ENTRYPOINT ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
HEALTHCHECK CMD [ "supervisorctl", "status " ]
EXPOSE 8080
