# The main purpose of this container is to run Stork Environment Simulator.
ARG KEA_REPO=public/isc/kea-2-4
ARG KEA_VERSION=2.4.0-isc20230630120747

# The demo setup is not fully compatible with arm64 architectures.
# In particular, only the amd64 image with named is available.
# In addition, the flamethrower program requires an older Debian
# version for which we provide no arm64 packages with perfdhcp.
# Since we use common containers for building Stork, building
# BIND9, Kea and simulator on different architectures is impossible.
# The good news is that amd64 can be emulated on top of the arm64.
FROM --platform=linux/amd64 debian:12.1-slim AS base

# Stage to compile Flamethrower.
# Flamethrower doesn't compile on Debian 12.1, so we use Debian 11 instead.
FROM --platform=linux/amd64 debian:bullseye-slim AS flamethrower-builder
# Install Flamethrower dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++=4:10.2.* \
        cmake=3.18.* \
        make=4.* \
        libldns-dev=1.7.* \
        libnghttp2-dev=1.43.* \
        libuv1-dev=1.40.* \
        libgnutls28-dev=3.7.* \
        pkgconf=1.7.* \
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
    && rm -rf /src
WORKDIR /app

# Stage to build the simulator.
FROM base AS simulator-builder
WORKDIR /app
# Install Python dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
        python3=3.11.* \
        python3-pip=23.* \
        python3-setuptools=66.* \
        python3-wheel=0.38.* \
        python3-venv=3.11.* \
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
    # Install curl.
    apt-get update && apt-get install -y --no-install-recommends \
        # Install curl.
        curl=7.* \
        ca-certificates=20230311 \
        gnupg=2.2.* \
        apt-transport-https=2.6.* \
    # Configure the ISC repository.
    && curl -1sLf "https://dl.cloudsmith.io/${KEA_REPO}/cfg/setup/bash.deb.sh" | bash \
    # Install runtime dependencies.
    && apt-get update && apt-get install -y --no-install-recommends \
        # Flamethrower dependencies.
        libldns3=1.8.* \
        libuv1=1.44.* \
        nghttp2=1.52.* \
        # Dig dependencies.
        dnsutils=1:9.18.* \
        # Kea Perfdhcp dependencies.
        isc-kea-perfdhcp=${KEA_VERSION} \
        isc-kea-common=${KEA_VERSION} \
        # Simulator dependencies.
        python3=3.11.* \
        python3-venv=3.11.* \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
# Copy the Flamethrower binary from the builder stage.
COPY --from=flamethrower-builder /app/flame /usr/local/bin/flame
# Copy the simulator source code and virtual environment from the builder stage.
WORKDIR /app
COPY --from=simulator-builder /app /app
# Start the simulator.
ENV FLASK_APP=sim.py
ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8
CMD ["/app/venv/bin/python3", "-m", "gunicorn", "-w", "4", "-t", "60", "-b", "0.0.0.0:5000", "sim:app"]
