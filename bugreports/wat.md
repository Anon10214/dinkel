When running the following query:
```cypher
OPTIONAL MATCH ()-[{EnrBRD:(""), EnrBRD:(1), EnrBRD:("q")}]->(Sm4:VuoRYb)<-[vthiLy:{EnrBRD:(false), EnrBRD:(""), EnrBRD:(true)}]-(:VuoRYb) MATCH ({OCuCGO:(0 + ((null)))}), (:nv:nv:nv:nv:nv)  UNWIND (CASE size(([]+(([]+null)+((null+null)+[])))) WHEN 1 THEN ([]+(([]+null)+((null+null)+[]))) ELSE [("")] END) AS tSl5 WITH (([])+[]) AS VEG3Qh7lqm ORDER BY (0)  SKIP (450864730787809894)  RETURN DISTINCT (0) AS WUBG  SKIP (6706851257924841501) 
```

I encountered the following error:
```
errMsg: Invalid input '{': expected a relationship type line: 1, column: 84, offset: 83 errCtx: ...nrBRD:("q")}]->(Sm4:VuoRYb)<-[vthiLy:{EnrBRD:(false), EnrBRD:(""), EnrBRD:... errCtxOffset: 40
```

I believe the query mentioned above is semantically and syntactically correct and thus no error should be thrown here.

I encountered this issue when testing queries on the **FalkorDB master branch** in a Docker container running **redis:7.2.3-bookworm**.

### Steps to reproduce
Run the following queries and observe it throws an error:
```cypher
MATCH ({OCuCGO:(null)}), (:nv:nv:nv:nv:nv)  WITH [] AS VEG3Qh7lqm ORDER BY (0)  SKIP (450864730787809894)  RETURN DISTINCT (0) AS WUBG  SKIP (6706851257924841501) 
---
OPTIONAL MATCH ()-[{EnrBRD:(""), EnrBRD:(1), EnrBRD:("q")}]->(Sm4:VuoRYb)<-[vthiLy:{EnrBRD:(false), EnrBRD:(""), EnrBRD:(true)}]-(:VuoRYb) MATCH ({OCuCGO:(0 + ((null)))}), (:nv:nv:nv:nv:nv)  UNWIND (CASE size(([]+(([]+null)+((null+null)+[])))) WHEN 1 THEN ([]+(([]+null)+((null+null)+[]))) ELSE [("")] END) AS tSl5 WITH (([])+[]) AS VEG3Qh7lqm ORDER BY (0)  SKIP (450864730787809894)  RETURN DISTINCT (0) AS WUBG  SKIP (6706851257924841501) 
```

### Expected behavior
The query should run successfully

### Actual behavior
The query fails with the error message `errMsg: Invalid input '{': expected a relationship type line: 1, column: 84, offset: 83 errCtx: ...nrBRD:("q")}]->(Sm4:VuoRYb)<-[vthiLy:{EnrBRD:(false), EnrBRD:(""), EnrBRD:... errCtxOffset: 40`