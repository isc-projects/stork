# The main purpose of this container is to run Stork Environment Simulator.
ARG KEA_REPO=public/isc/kea-2-4
ARG KEA_VERSION=2.4.0-isc20230630120747

FROM debian:12.1-slim AS base
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        # Install curl.
        curl \
        ca-certificates \
        gnupg \
        apt-transport-https \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Stage to compile Flamethrower.
# Flamethrower doesn't compile on Debian 12.1, so we use Debian 11 instead.
FROM debian:bullseye-slim AS flamethrower-builder
# Install Flamethrower dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++ \
        cmake \
        make \
        libldns-dev \
        libnghttp2-dev \
        libuv1-dev \
        libgnutls28-dev \
        pkgconf \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
# Directory for the compiled binary.
WORKDIR /app
# Directory for the source files.
WORKDIR /src
# Fetch the Flamethrower source code.
ADD https://github.com/DNS-OARC/flamethrower/archive/refs/tags/v0.11.0.tar.gz flamethrower.tar.gz
WORKDIR /src/build
RUN \
    # Extract the archive.
    tar -xzf /src/flamethrower.tar.gz --strip-components=1 -C /src \
    # Configure the build.
    && cmake -DDOH_ENABLE=ON -DCMAKE_BUILD_TYPE=RelWithDebInfo /src \
    # Compile the binary.
    && make \
    # Copy the binary to the /app directory.
    && cp flame /app/flame \
    # Cleanup.
    && cd / \
    && rm -rf /src

# Stage to build the simulator.
FROM base AS simulator-builder
WORKDIR /app
# Install Python dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
        python3 \
        python3-pip \
        python3-setuptools \
        python3-wheel \
        python3-venv \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    # Create a virtual environment.
    && python3 -m venv venv
# Copy the simulator requirements.
COPY tests/sim/requirements.txt .
RUN \
    # Activate the virtual environment.
    . venv/bin/activate \
    # Install the simulator dependencies.
    && pip3 install --no-cache-dir -r requirements.txt
# Copy rest of the simulator source code.
COPY tests/sim .

# Stage to run the simulator.
FROM base AS runner
ARG KEA_REPO
ARG KEA_VERSION
RUN \
    # Configure the ISC repository.
    curl -1sLf "https://dl.cloudsmith.io/${KEA_REPO}/cfg/setup/bash.deb.sh" | bash \
    # Install runtime dependencies.
    && apt-get update && apt-get install -y --no-install-recommends \
        # Flamethrower dependencies.
        libldns3 \
        libuv1 \
        nghttp2 \
        # Dig dependencies.
        dnsutils \
        # Kea Perfdhcp dependencies.
        isc-kea-perfdhcp=${KEA_VERSION} \
        # Simulator dependencies.
        python3 \
        python3-venv \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
# Copy the Flamethrower binary from the builder stage.
COPY --from=flamethrower-builder /app/flame /usr/local/bin/flame
# Copy the simulator source code and virtual environment from the builder stage.
WORKDIR /sim
COPY --from=simulator-builder /app /sim
# Start the simulator.
ENV FLASK_ENV=development
ENV FLASK_APP=sim.py
ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8
CMD ["/sim/venv/bin/python3", "-m", "flask", "run", "--host", "0.0.0.0" ]
