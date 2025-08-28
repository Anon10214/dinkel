FROM maven:3.9.6-eclipse-temurin-21-alpine

RUN apk add perl git unzip

# Install jacoco
ADD https://search.maven.org/remotecontent?filepath=org/jacoco/jacoco/0.8.12/jacoco-0.8.12.zip jacoco.zip
RUN unzip jacoco.zip -d jacoco

# Clone repo
RUN git clone --depth 1 -b 5.6 https://github.com/neo4j/neo4j.git

WORKDIR /neo4j

# Set the right java version
RUN rm -rf /opt/java/openjdk
RUN export VERSION=$(sed -n 's/.*<maven\.compiler\.target>\(\d\+\).*$/\1/p' pom.xml); apk add openjdk${VERSION}; ln -s /usr/lib/jvm/java-${VERSION}-openjdk /opt/java/openjdk

# Remove spotless dependency
RUN perl -i -0pe 's#<plugin>\s*<groupId>com.diffplug.spotless.*?</plugin>##gs' pom.xml
# Remove license check and replace it with jacoco
RUN perl -i -0pe 's#<plugin>\s*<groupId>com.mycila</groupId>\s*<artifactId>license-maven-plugin.*?</plugin>##gs' pom.xml
RUN perl -i -0pe 's#<plugin>\s*<groupId>org.neo4j.build.plugins</groupId>\s*<artifactId>licensing-maven-plugin.*?</plugin>##gs' pom.xml

# Build
# sbt.io.jdktimestamps flag because of https://github.com/sbt/sbt/issues/7463#issuecomment-1856824967
RUN mvn clean install -DskipTests -T1C -Dsbt.io.jdktimestamps=true

RUN mkdir out
RUN tar xzvf packaging/standalone/target/neo4j-community-*-SNAPSHOT-unix.tar.gz -C out --strip-components=1

ADD https://github.com/neo4j/apoc/releases/download/5.6.0/apoc-5.6.0-core.jar out/plugins

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

# Use jacoco
RUN echo 'server.jvm.additional=-javaagent:/jacoco/lib/jacocoagent.jar=output=tcpserver' >> out/conf/neo4j.conf
# Setup report script
RUN mkdir /neo4j-cov
RUN printf \
	"#!/bin/sh\n \
	cur_time=\$(date +%%s)\n \
	mkdir -p /neo4j-cov/\$cur_time\n \
	ln -nfs /neo4j-cov/\${cur_time} /neo4j-cov/latest\n \
	java -jar /jacoco/lib/jacococli.jar dump --destfile /neo4j-cov/\${cur_time}_dump\n \
	java -jar /jacoco/lib/jacococli.jar report --html /neo4j-cov/\$cur_time /neo4j-cov/\${cur_time}_dump \$(ls -d /neo4j/out/lib/* | \
	# Remove jars that are already loaded by java. Not excluding those causes jacoco to fail
	grep -v -e cypher-shell -e jakarta.xml -e jersey-common -e log4j-api -e log4j-core -e neo4j-bootcheck | \
	xargs printf '--classfiles %%s ')\n \
	tar czf /neo4j-cov/\${cur_time}.tar.gz -P --transform s#/neo4j-cov/\${cur_time}#coverage# /neo4j-cov/\${cur_time}\n \
	ln -nfs /neo4j-cov/\${cur_time}.tar.gz /neo4j-cov/latest.tar.gz" > /neo4j-cov/get-cov
RUN sed -i 's#--classfiles /neo4j/out/lib/neo4j-bootcheck-\([0-9]\+\.\)\{2\}[0-9]\+-SNAPSHOT\.jar ##' /neo4j-cov/get-cov
RUN chmod +x /neo4j-cov/get-cov

CMD sh -c "apk add tmux && tmux new -d 'timeout 48h sh -c \"while true; do /neo4j-cov/get-cov && sleep 5m; done\"' && sh -c 'while true; do timeout -s SIGKILL 1h out/bin/neo4j console; done'"
