FROM debian:bookworm AS builder

ENV VERSION=b8ec5ca67296565b7fa226f878a64ecb32b8a68c

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

RUN git reset --hard $VERSION

RUN git submodule update --init --recursive

ENV ASAN_OPTIONS=detect_leaks=0

RUN . "$HOME/.cargo/env" && make SAN=address

FROM redis:7.2.3-bookworm

WORKDIR /app

COPY --from=builder /app/bin/linux-x64-debug-asan/src/falkordb.so .

RUN apt-get update \
    && apt-get install -y gcc libomp-dev

EXPOSE 6379

# Run the falkordb-server, preloading asan and loading falkordb
# Putting it in an infinite loop so it restarts on a crash
CMD while true; do LD_PRELOAD=$(gcc -print-file-name=libasan.so) redis-server --loadmodule ./falkordb.so --save ""; done