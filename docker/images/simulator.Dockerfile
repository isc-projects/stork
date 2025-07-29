# The main purpose of this container is to run Stork Environment Simulator.
ARG KEA_REPO=public/isc/kea-dev
ARG KEA_VERSION=2.7.8-isc20250429105336
ARG FLAMETHROWER_COMMIT=0ee1fba170d7673e32dc0226d08732fd08acc7ac

FROM debian:12.1-slim AS base

# Stage to compile Flamethrower.
FROM debian:12.1-slim AS flamethrower-builder
ARG FLAMETHROWER_COMMIT
# Install Flamethrower dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
        g++=4:12.* \
        cmake=3.25.* \
        make=4.* \
        libldns-dev=1.8.* \
        libnghttp2-dev=1.52.* \
        libuv1-dev=1.44.* \
        libgnutls28-dev=3.7* \
        pkgconf=1.8.* \
        unzip=6.* \
    # Cleanup.
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
# Directory for the compiled binary.
WORKDIR /app
# Directory for the source files.
WORKDIR /src
# Fetch the Flamethrower source code.
ADD https://github.com/DNS-OARC/flamethrower/archive/${FLAMETHROWER_COMMIT}.zip flamethrower.zip
WORKDIR /src/build
RUN \
    # Extract the archive.
    unzip /src/flamethrower.zip -d /src \
    # Configure the build.
    && cmake -DDOH_ENABLE=ON -DCMAKE_BUILD_TYPE=RelWithDebInfo /src/flamethrower-${FLAMETHROWER_COMMIT} \
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
CMD ["/app/venv/bin/python3", "-m", "gunicorn", "-w", "1", "-t", "60", "-b", "0.0.0.0:5000", "sim:app"]
