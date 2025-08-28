package bisect

import (
	"time"

	"github.com/CelineWuest/biscepter/pkg/biscepter"
	"github.com/sirupsen/logrus"
)

// TODO: Add warning somewhere that graph fingerprinting is disabled during bisection, and to adjust RETURNs to make the bug apparent in returned rows
var configMap map[string]*biscepter.Job = map[string]*biscepter.Job{
	"neo4j": {
		Repository: "https://github.com/neo4j/neo4j.git",
		BuildCost:  50,
		Ports:      []int{7687},
		Healthchecks: []biscepter.Healthcheck{
			{
				Port:      7474,
				CheckType: biscepter.HttpGet200,
				Data:      "/",
				Config: biscepter.HealthcheckConfig{
					Retries:          30,
					Backoff:          time.Second,
					BackoffIncrement: time.Second / 2,
					MaxBackoff:       5 * time.Second,
				},
			},
		},
		GoodCommit: "1104da8a62c763122a27904e6965c8d05461fd31", // Oldest commit of Neo4j 4.0
		BadCommit:  "c68156edf24164435ab1ac257ec633134c2887f7", // Newest commit of Neo4j 5.17
		Log:        logrus.StandardLogger(),
		Dockerfile: `
FROM maven:3.9.6-eclipse-temurin-21-alpine

RUN apk add perl

WORKDIR /app
COPY . .

# Set the right java version
RUN rm -rf /opt/java/openjdk
RUN export VERSION=$(sed -n 's/vm\.target\.version/maven.compiler.target/g;s/.*<maven\.compiler\.target>\(\d\+\).*$/\1/p' pom.xml); apk add openjdk${VERSION}; ln -s /usr/lib/jvm/java-${VERSION}-openjdk /opt/java/openjdk

# Remove spotless dependency
RUN perl -i -0pe 's#<plugin>\s*<groupId>com.diffplug.spotless.*?</plugin>##gs' pom.xml
# Remove license check
RUN perl -i -0pe 's#<plugin>\s*<groupId>com.mycila</groupId>\s*<artifactId>license-maven-plugin.*?</plugin>##gs' pom.xml
RUN perl -i -0pe 's#<plugin>\s*<groupId>org.neo4j.build.plugins</groupId>\s*<artifactId>licensing-maven-plugin.*?</plugin>##gs' pom.xml

# Build
# sbt.io.jdktimestamps flag because of https://github.com/sbt/sbt/issues/7463#issuecomment-1856824967
RUN mvn clean install -DskipTests -T1C -Dsbt.io.jdktimestamps=true

# Extract results
RUN mkdir out
RUN tar xzvf packaging/standalone/target/neo4j-*-unix.tar.gz -C out --strip-components=1

# Disable auth
RUN sed -i "s/#dbms.security.auth_enabled=false/dbms.security.auth_enabled=false/g" out/conf/neo4j.conf
# Allow non-local connections (different versions have different names for the config)
RUN sed -i "s/#server.default_listen_address=0.0.0.0/server.default_listen_address=0.0.0.0/g" out/conf/neo4j.conf
RUN sed -i "s/#dbms.default_listen_address=0.0.0.0/dbms.default_listen_address=0.0.0.0/g" out/conf/neo4j.conf
RUN sed -i "s/#dbms.connectors.default_listen_address=0.0.0.0/dbms.connectors.default_listen_address=0.0.0.0/g" out/conf/neo4j.conf
# Disable restriction on procedures
RUN sed -i "s/#dbms.security.procedures.unrestricted=my.extensions.example,my.procedures.*/dbms.security.procedures.unrestricted=*/g" out/conf/neo4j.conf

EXPOSE 7687
EXPOSE 7474


# Checking if the server starts up without crashing (plugin loads may crash the server but the build can succeed)
# timeout exits with code 137 if timeout sent the KILL signal
RUN timeout -s KILL 15s ./out/bin/neo4j console; test $? -eq 137

# Make sure neo4j is no longer running
RUN out/bin/neo4j stop

CMD out/bin/neo4j console
`,
	},
	"falkordb": {
		Repository: "https://github.com/FalkorDB/FalkorDB.git",
		// TODO: Investigate buildcost
		BuildCost: 50,
		Ports:     []int{6379},
		Healthchecks: []biscepter.Healthcheck{
			{
				Port:      6379,
				CheckType: biscepter.Script,
				// Send PING to server using nc or ncat, expecting response of PONG
				Data: `echo PING | nc localhost $PORT6379 -q 1 | grep -q +PONG || echo PING | ncat localhost $PORT6379 -w 1 | grep -q +PONG`,
				Config: biscepter.HealthcheckConfig{
					Retries: 10,
					Backoff: time.Second / 2,
				},
			},
		},
		GoodCommit: "80e5102d4eb00f465d47f1ed3921e5dba0b6a04c", // Last commit before v2.0.0 release
		BadCommit:  "da0b71d1c31fef53bb659e2687a79623d10bfa05", // Newest commit as of 04.04.2024
		Log:        logrus.StandardLogger(),
		Dockerfile: `
FROM debian:bookworm AS builder

WORKDIR /app

# Install rust
RUN apt-get update --allow-releaseinfo-change && \
    apt-get install -y git build-essential cmake m4 automake peg libtool autoconf python3 python3-pip clang cargo curl libssl-dev libomp-dev

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

RUN . "$HOME/.cargo/env" && \
    rustup install nightly && \
    rustup default nightly

# Copy the source code
COPY . .

# Older versions have multiple definitions it seems, allow that
ENV LDFLAGS="-z muldefs"
# Ignore warnings
ENV CFLAGS="-Wno-error"

# Remove all annoying diagnostic pragmas causing errors
RUN grep -lr "pragma GCC diagnostic error" . | xargs sed -i "s/.*pragma GCC diagnostic error.*//g"

# Old readies defaults to python2, set MK.pyver to 3 instead
RUN . "$HOME/.cargo/env" && mkdir -p bin/linux-x64-release/src && make clean; make CLANG=1 GCC=0 MK.pyver:=3

RUN mv bin/linux-x64-release/src/falkordb.so falkordb.so || mv /app/src/redisgraph.so falkordb.so || mv bin/linux-x64-release/src/redisgraph.so falkordb.so

FROM redis:7.2.4-bookworm

WORKDIR /app

# For libomp
RUN apt-get update \
    && apt-get install -y clang libomp-dev build-essential wget

RUN wget http://download.redis.io/releases/redis-5.0.9.tar.gz && wget http://download.redis.io/releases/redis-6.2.9.tar.gz
RUN tar xzf redis-5.0.9.tar.gz && tar xzf redis-6.2.9.tar.gz

RUN cd redis-5.0.9 && make
RUN cd redis-6.2.9 && make

COPY --from=builder /app/falkordb.so .

EXPOSE 6379

# Run the falkordb-server
# Putting it in an infinite loop so it restarts on a crash
CMD while true; do \
	redis-server --loadmodule ./falkordb.so --save "" --protected-mode no || \
	./redis-5.0.9/src/redis-server --loadmodule ./falkordb.so --save "" --protected-mode no || \
	./redis-6.2.9/src/redis-server --loadmodule ./falkordb.so --save "" --protected-mode no    \
	; done
`,
	},
}
