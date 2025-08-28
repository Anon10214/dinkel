I found a bug using my cypher fuzzer.

When running the following query against an empty database:
```cypher
MATCH (YoapzB{ya6:("a")}) WHERE (false) WITH DISTINCT *  SKIP (1) LIMIT (5798942283600013095) OPTIONAL MATCH (n6{ya6:(""), ya6:("")})-[NpClphp{ya6:("WV"), ya6:last([(false)])}]->(W3QiQyzG:x0GiP)<-[]-(:x0GiP{ya6:(0)}) WHERE (true) CREATE (m6vY568:x0GiP)-[CgUAVfq:uu{hz:((-1284085925319075551)%(1)), hz:(-7341629386614728049)}]->(yWjWYyU:x0GiP) SET yWjWYyU.ya6 = (4657269650196687458) MERGE (tayPqPFbRA:x0GiP{LYr:(""), Kn6:(8.001240709350745e-26)}) RETURN (-6.882228226640261e-40) AS RDTU8XeL UNION ALL  RETURN avg((-3.056682629547176e+108)) AS RDTU8XeL
```

I encountered the following error:
```
pq: UNION types agtype and double precision cannot be matched
```

I believe the query mentioned above is semantically and syntactically correct and thus no error should be thrown here.

Additionally, the query runs successfully in neo4j.

I encountered this issue when testing queries against the **apache/age:PG13_latest** docker image.

### Steps to reproduce

Spin up a local instance of **apache/age:PG13_latest**: `docker run -e POSTGRES_PASSWORD=123 --rm --name age apache/age:PG13_latest`

Get a shell in the docker container: `docker exec -it age /bin/bash`

Connect to postgres: `su postgres -c psql`

Run the following queries:
```psql
LOAD 'age';
---
SET search_path = ag_catalog, "$user", public;
---
SELECT create_graph('graph');
---
SELECT * FROM cypher('graph',$$
	MATCH (YoapzB{ya6:("a")}) WHERE (false) WITH DISTINCT *  SKIP (1) LIMIT (5798942283600013095) OPTIONAL MATCH (n6{ya6:(""), ya6:("")})-[NpClphp{ya6:("WV"), ya6:last([(false)])}]->(W3QiQyzG:x0GiP)<-[]-(:x0GiP{ya6:(0)}) WHERE (true) CREATE (m6vY568:x0GiP)-[CgUAVfq:uu{hz:((-1284085925319075551)%(1)), hz:(-7341629386614728049)}]->(yWjWYyU:x0GiP) SET yWjWYyU.ya6 = (4657269650196687458) MERGE (tayPqPFbRA:x0GiP{LYr:(""), Kn6:(8.001240709350745e-26)}) RETURN (-6.882228226640261e-40) AS RDTU8XeL UNION ALL  RETURN avg((-3.056682629547176e+108)) AS RDTU8XeL
$$) as (v agtype);
---
MATCH (YoapzB{ya6:("a")}) WHERE (false) WITH DISTINCT *  SKIP (1) LIMIT (5798942283600013095) OPTIONAL MATCH (n6{ya6:(""), ya6:("")})-[NpClphp{ya6:("WV"), ya6:last([(false)])}]->(W3QiQyzG:x0GiP)<-[]-(:x0GiP{ya6:(0)}) WHERE (true) CREATE (m6vY568:x0GiP)-[CgUAVfq:uu{hz:((-1284085925319075551)%(1)), hz:(-7341629386614728049)}]->(yWjWYyU:x0GiP) SET yWjWYyU.ya6 = (4657269650196687458) MERGE (tayPqPFbRA:x0GiP{LYr:(""), Kn6:(8.001240709350745e-26)}) RETURN (-6.882228226640261e-40) AS RDTU8XeL UNION ALL  RETURN avg((-3.056682629547176e+108)) AS RDTU8XeL
```

### Expected behavior
The query should run successfully

### Actual behavior
The query fails with the error message `pq: UNION types agtype and double precision cannot be matched`.
