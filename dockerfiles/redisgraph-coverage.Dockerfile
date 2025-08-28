FROM redis:7.0.11-bullseye

WORKDIR /app

# Setup report script
RUN mkdir /redisgraph-cov
RUN printf \
    "#!/bin/sh\n \
    cur_time=\$(date +%%s)\n \
    mkdir -p /redisgraph-cov/\$cur_time\n \
    ln -nfs /redisgraph-cov/\${cur_time} /redisgraph-cov/latest\n \
    kill -10 \$(redis-cli INFO | grep process_id | awk -F ':' '{print \$2+0}')\n \
    lcov --rc lcov_branch_coverage=1 -c -d /app --output-file /redisgraph-cov/latest/cov.info --include '/app/*'\n \
    lcov --rc lcov_branch_coverage=1 --summary /redisgraph-cov/latest/cov.info | tee /redisgraph-cov/latest/summary.txt" > /redisgraph-cov/get-cov
RUN chmod +x /redisgraph-cov/get-cov

RUN apt-get update \
    && apt-get install -y git clang

# Clone the main branch
RUN git clone --recurse-submodules --depth 1 -b 2.12 -j8 https://github.com/RedisGraph/RedisGraph.git .

# Register signal handler for SIGUSR1
RUN sed -i '/OnLoad/i #include <signal.h>\n \
    void __gcov_flush(void); \
    void sig_handler(int signum){printf("Flushing gcov\\n");__gcov_flush();}' src/module.c
RUN sed -i '/OnLoad/a \
    struct sigaction action; \
    action.sa_handler = sig_handler; \
    sigemptyset(&action.sa_mask); \
    action.sa_flags = 0; \
    sigaction(SIGUSR1, &action, NULL);' src/module.c

# Flush gcov data on redis crash
RUN sed -i '/void InfoFunc/i void __gcov_flush(void);' src/debug.c
RUN sed -i '/if(!for_crash_report) return;/a __gcov_flush();' src/debug.c

RUN ./sbin/setup

RUN make COV=1

EXPOSE 6379

# Run the redis-server
# Putting it in an infinite loop so it restarts on a crash
# Send a SIGUSR1 signal to the process every minute to flush the gcov data
CMD bash -c "while true; do kill -10 \$(redis-cli INFO | grep process_id | awk -F ':' '{print \$2+0}'); sleep 1m; done &"; apt update && apt install tmux -y && tmux new -d 'timeout 48h sh -c "while true; do /redisgraph-cov/get-cov && sleep 5m; done"' && while true; do timeout -s SIGKILL 1h nice -n 5 redis-server --loadmodule /app/bin/linux-x64-debug-cov/src/redisgraph.so --save ""; done