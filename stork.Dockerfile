ARG DEBIAN_VER=11.2-slim

FROM debian:${DEBIAN_VER} AS debian-base
ENV CI=true

# Install system-wide dependencies
FROM debian-base AS base
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
        # System-wise dependencies
        && apt-get install \
                -y \
                --no-install-recommends \
                ca-certificates=20210119 \
                wget=1.21-1+deb11u1 \
                unzip=6.0-26 \
                ruby-dev=1:2.7+2 \
                python3.9=3.9.2-1 \
                python3-pip=20.3.4-4 \
                make=4.3-4.1 \
                gcc=4:10.2.1-1 \
                xz-utils=5.2.5-2 \
                libc6-dev=2.31-13+deb11u2 \
                ruby-rubygems=3.2.5-2 \
                openjdk-11-jre-headless=11.0.14+9-1~deb11u1 \
                chromium=99.0.4844.51-1~deb11u1 \
                git=1:2.30.2-1 \
        && apt-get clean \
        && rm -rf /var/lib/apt/lists/*

# Install main dependencies
FROM base AS prepare 
WORKDIR /app/rakelib
COPY rakelib/1_init.rake ./
WORKDIR /app/rakelib/init_debs
COPY rakelib/init_debs ./
WORKDIR /app
COPY Rakefile ./
RUN rake prepare_env

# Backend dependencies installation
FROM prepare AS gopath-prepare
WORKDIR /app/rakelib
COPY rakelib/2_codebase.rake ./
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN rake prepare_backend_deps

# Frontend dependencies installation
FROM prepare AS nodemodules-prepare
WORKDIR /app/rakelib
COPY rakelib/2_codebase.rake ./
WORKDIR /app/webui
COPY webui/package.json webui/package-lock.json ./
RUN rake prepare_ui_deps

# General-purpose stage for tasks: building, testing, linting, etc.
# It contains the codebase with dependencies
FROM prepare AS codebase
WORKDIR /app/tools/golang
COPY --from=gopath-prepare /app/tools/golang .
WORKDIR /app/webui
COPY --from=nodemodules-prepare /app/webui .
WORKDIR /app
COPY . .

# Build the Stork binaries
# It builds all components at once.
FROM codebase AS builder
RUN rake build_server_dist build_agent_dist

# Minimal agent container
FROM debian-base as agent
COPY --from=builder /app/dist/agent /
ENTRYPOINT [ "/usr/bin/stork-agent" ]

# Minimal server container
FROM debian-base AS server
COPY --from=builder /app/dist/server /
ENTRYPOINT [ "/usr/bin/stork-server" ]
