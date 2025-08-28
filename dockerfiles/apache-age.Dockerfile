# WARNING:
# If you want to use gdb to get the stack traces of crashes, run this image with --cap-add=SYS_PTRACE
# Then, open a second shell in the container once spun up and run gdb, then execute the following instructions:
# - set follow-fork-mode child
# - attach 1
# - c
# Now, execute the crashing query. Once postgres segfaults, run bt in gdb to get the stack trace
FROM apache/age:release_PG16_1.5.0

RUN apt update && apt install -y gdb

ENV POSTGRES_HOST_AUTH_METHOD=trust
