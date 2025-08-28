FROM debian:bookworm AS builder

# Build falkordb
WORKDIR /app

# Install rust
RUN apt-get update --allow-releaseinfo-change && \
    apt-get install -y git build-essential cmake m4 automake peg libtool autoconf python3 python3-pip clang cargo curl libssl-dev libomp-dev

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

RUN . "$HOME/.cargo/env" && \
    rustup install nightly && \
    rustup default nightly

# Clone the main branch
RUN git clone --recurse-submodules -j8 https://github.com/FalkorDB/FalkorDB.git .

RUN . "$HOME/.cargo/env" && make deps CLANG=1 GCC=0

ARG VERSION=9a3de5d06e847e19caed99d256ca1d1d0d92f9a4

# Grab new refs
RUN git fetch origin

RUN git reset --hard $VERSION

RUN git submodule update --init --recursive -f

RUN . "$HOME/.cargo/env" && make CLANG=1 GCC=0 SAN=address

FROM redis:7.2.3-bookworm

WORKDIR /app

# For libomp
RUN apt-get update \
    && apt-get install -y gcc clang libomp-dev

COPY --from=builder /app/bin/linux-x64-debug-asan/src/falkordb.so .

EXPOSE 6379

# Run the falkordb-server
# Putting it in an infinite loop so it restarts on a crash
CMD while true; do LD_PRELOAD=$(gcc -print-file-name=libasan.so) redis-server --loadmodule ./falkordb.so CACHE_SIZE 1 --save ""; done
