I found a bug using my cypher fuzzer.

When running the following query against an empty database:
```cypher
MATCH () MERGE x = (:C)
```

I encountered the following error:
```
pq: a path is of the form: [vertex, (edge, vertex)*i] where i >= 0
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
	MATCH () MERGE x = (:C)
$$) as (v agtype);
---
MERGE ()-[:A]->()-[:B]->()
---
MATCH () MERGE x = (:C)
```

### Expected behavior
The query should run successfully

### Actual behavior
The query fails with the error message `pq: a path is of the form: [vertex, (edge, vertex)*i] where i >= 0`.
