FROM debian:bullseye AS builder

SHELL ["/bin/bash", "-c"]

RUN apt-get update && apt-get install -y git

ADD https://s3-eu-west-1.amazonaws.com/deps.memgraph.io/toolchain-v4/toolchain-v4-binaries-debian-11-amd64.tar.gz toolchain.tar.gz

RUN tar xzvfm toolchain.tar.gz -C /opt \
    && rm toolchain.tar.gz

RUN git clone --depth 1 --branch release/2.15.1 https://github.com/memgraph/memgraph.git

WORKDIR /memgraph

# Install toolchain build deps
RUN ./environment/os/debian-11.sh install TOOLCHAIN_BUILD_DEPS

# Install toolchain deps
RUN ./environment/os/debian-11.sh install TOOLCHAIN_RUN_DEPS

# Install memgraph build deps
RUN ./environment/os/debian-11.sh install MEMGRAPH_BUILD_DEPS

# Every command from here on out must run in the build environment, so they are prepended by a source command

# Activate the toolchain
RUN source /opt/toolchain-v4/activate && ./init

RUN mkdir -p build

RUN source /opt/toolchain-v4/activate && cd build \
    # Activate address sanitizer
    && cmake -DASAN=ON .. \
    # Build memgraph
    && make -j$(nproc) memgraph

CMD build/memgraph --storage-properties-on-edges=true
