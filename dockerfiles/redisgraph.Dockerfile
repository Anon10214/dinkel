FROM redis:7.0.11-bullseye

ENV VERSION=4646ed7012609aba60f32b38bb4b2fee09d9c6fe

# Build redisgraph
WORKDIR /app

RUN apt-get update \
    && apt-get install -y git clang

# Clone the main branch
RUN git clone --recurse-submodules -j8 https://github.com/RedisGraph/RedisGraph.git .

RUN git reset --hard $VERSION

RUN ./sbin/setup

RUN make

EXPOSE 6379

# Run the redis-server
# Putting it in an infinite loop so it restarts on a crash
CMD while true; do redis-server --loadmodule /app/bin/linux-x64-release/src/redisgraph.so --save ""; done