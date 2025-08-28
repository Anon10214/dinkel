FROM eclipse-temurin:17-jdk-alpine

ARG VERSION=5.26.2

ADD https://dist.neo4j.org/neo4j-enterprise-$VERSION-unix.tar.gz neo4j.tar.gz
RUN tar -xzf neo4j.tar.gz \
    && rm neo4j.tar.gz

WORKDIR /neo4j-enterprise-$VERSION

# Move the APOC library to plugins to enable it
RUN cp labs/apoc-*-core.jar plugins

# Disable auth
RUN sed -i "s/#dbms.security.auth_enabled=false/dbms.security.auth_enabled=false/g" conf/neo4j.conf
# Allow non-local connections
RUN sed -i "s/#server.default_listen_address=0.0.0.0/server.default_listen_address=0.0.0.0/g" conf/neo4j.conf
# Disable browser, as we only access the DBMS via the bolt console
RUN sed -i "s/server.http.enabled=true/server.http.enabled=false/g" conf/neo4j.conf
# Disable restriction on procedures
RUN sed -i "s/#dbms.security.procedures.unrestricted=my.extensions.example,my.procedures.*/dbms.security.procedures.unrestricted=*/g" conf/neo4j.conf

# Accept license
RUN bin/neo4j-admin server license --accept-evaluation

EXPOSE 7687

CMD bin/neo4j console --verbose
