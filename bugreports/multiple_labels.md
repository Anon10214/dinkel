I found a bug using my cypher fuzzer.

When running the following query against an empty database:
```cypher
 CREATE iRJUb4 = (Qrft:YAviJFXeY)<-[:ctGCVk]-(eia9sE:YAviJFXeY)<-[:ctGCVk{H:null, H:(9223372036854775807)}]-(eia9sE)<-[R67Zz:ctGCVk{H:(8195962006749411239), Z:(""), Z:(null)}]-(cW:YAviJFXeY)<-[:ctGCVk]-(Qrft) MERGE (qSCHa:YAviJFXeY{Z:(-1)})-[UMf:ctGCVk]->(:YAviJFXeY)<-[:ctGCVk{qM8O:(9223372036854775807), Z:(5312773935170032850), Z:(1375658972416289474)}]-(c:YAviJFXeY)-[:ctGCVk]->(:YAviJFXeY{mxvjaJ:(8.824797939473958e+60)})-[:rR]->(_:YAviJFXeY)<-[:ctGCVk]-(eia9sE)-[:rR{Z:(4.6100868439768537e-212)}]->(qSCHa)-[f:ctGCVk]->(Qrft)-[:rR{qM8O:(9223372036854775807), Z:(true)}]->(BPPm:YAviJFXeY)-[lS:ctGCVk]->(Qrft)-[:ctGCVk{rzuI5fJc:(true), nS:(false)}]->(cW) RETURN DISTINCT *  SKIP (3218351405375140871) 
```

I encountered the following error:
```
pq: multiple labels for variable 'Qrft' are not supported
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
	 CREATE iRJUb4 = (Qrft:YAviJFXeY)<-[:ctGCVk]-(eia9sE:YAviJFXeY)<-[:ctGCVk{H:null, H:(9223372036854775807)}]-(eia9sE)<-[R67Zz:ctGCVk{H:(8195962006749411239), Z:(""), Z:(null)}]-(cW:YAviJFXeY)<-[:ctGCVk]-(Qrft) MERGE (qSCHa:YAviJFXeY{Z:(-1)})-[UMf:ctGCVk]->(:YAviJFXeY)<-[:ctGCVk{qM8O:(9223372036854775807), Z:(5312773935170032850), Z:(1375658972416289474)}]-(c:YAviJFXeY)-[:ctGCVk]->(:YAviJFXeY{mxvjaJ:(8.824797939473958e+60)})-[:rR]->(_:YAviJFXeY)<-[:ctGCVk]-(eia9sE)-[:rR{Z:(4.6100868439768537e-212)}]->(qSCHa)-[f:ctGCVk]->(Qrft)-[:rR{qM8O:(9223372036854775807), Z:(true)}]->(BPPm:YAviJFXeY)-[lS:ctGCVk]->(Qrft)-[:ctGCVk{rzuI5fJc:(true), nS:(false)}]->(cW) RETURN DISTINCT *  SKIP (3218351405375140871) 
$$) as (v agtype);
---
 CREATE iRJUb4 = (Qrft:YAviJFXeY)<-[:ctGCVk]-(eia9sE:YAviJFXeY)<-[:ctGCVk{H:null, H:(9223372036854775807)}]-(eia9sE)<-[R67Zz:ctGCVk{H:(8195962006749411239), Z:(""), Z:(null)}]-(cW:YAviJFXeY)<-[:ctGCVk]-(Qrft) MERGE (qSCHa:YAviJFXeY{Z:(-1)})-[UMf:ctGCVk]->(:YAviJFXeY)<-[:ctGCVk{qM8O:(9223372036854775807), Z:(5312773935170032850), Z:(1375658972416289474)}]-(c:YAviJFXeY)-[:ctGCVk]->(:YAviJFXeY{mxvjaJ:(8.824797939473958e+60)})-[:rR]->(_:YAviJFXeY)<-[:ctGCVk]-(eia9sE)-[:rR{Z:(4.6100868439768537e-212)}]->(qSCHa)-[f:ctGCVk]->(Qrft)-[:rR{qM8O:(9223372036854775807), Z:(true)}]->(BPPm:YAviJFXeY)-[lS:ctGCVk]->(Qrft)-[:ctGCVk{rzuI5fJc:(true), nS:(false)}]->(cW) RETURN DISTINCT *  SKIP (3218351405375140871) 
```

### Expected behavior
The query should run successfully

### Actual behavior
The query fails with the error message `pq: multiple labels for variable 'Qrft' are not supported`.
