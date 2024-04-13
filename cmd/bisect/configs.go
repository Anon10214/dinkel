package bisect

import (
	"time"

	"github.com/Anon10214/dinkel/biscepter/pkg/biscepter"
	"github.com/sirupsen/logrus"
)

// TODO: Add warning somewhere that graph fingerprinting is disabled during bisection, and to adjust RETURNs to make the bug apparent in returned rows
var configMap map[string]*biscepter.Job = map[string]*biscepter.Job{
	"neo4j": {
		Repository: "https://github.com/neo4j/neo4j.git",
		BuildCost:  1000,
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
		BadCommit:  "b76dd4f846db9ba411e531d3e5c458614162c7ba", // Newest commit of Neo4j 5.17
		Log:        logrus.StandardLogger(),
		Dockerfile: `
FROM maven:3.9.6-eclipse-temurin-21-alpine

RUN apk add perl

WORKDIR /app
COPY . .

# Set the right java version
RUN rm -rf /opt/java/openjdk
RUN export VERSION=$(sed -n 's/.*<maven\.compiler\.target>\(\d\+\).*$/\1/p' pom.xml); apk add openjdk${VERSION}; ln -s /usr/lib/jvm/java-${VERSION}-openjdk /opt/java/openjdk

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
RUN tar xzvf packaging/standalone/target/neo4j-community-*-SNAPSHOT-unix.tar.gz -C out --strip-components=1

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
}
