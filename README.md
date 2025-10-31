<p align=center>
  <img alt="Dinkel Logo" height=200 src="https://user-images.githubusercontent.com/42354311/236577881-8817be72-37ff-4ae2-b347-0283f1daa0d4.svg"/>
</p>

<h2 align=center>The Powerful and Adaptable Cypher Fuzzer</h2>

<p align=center>
  <img alt="License Badge" src="https://img.shields.io/github/license/Anon10214/dinkel">
  <a href="https://pkg.go.dev/github.com/Anon10214/dinkel?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="Dinkel GoDoc"></a>
</p>

<p align=center>
  <a href="https://github.com/Anon10214/dinkel/actions/workflows/build.yml"><img alt="CI/CD Build Status Badge" src="https://github.com/Anon10214/dinkel/actions/workflows/build.yml/badge.svg"></a>
  <a href="https://github.com/Anon10214/dinkel/actions/workflows/test.yml"><img alt="CI/CD Test Status Badge" src="https://github.com/Anon10214/dinkel/actions/workflows/test.yml/badge.svg"></a>
  <a href="https://codecov.io/gh/Anon10214/dinkel"><img alt="CI/CD Coverage Status Badge" src="https://codecov.io/gh/Anon10214/dinkel/branch/main/graph/badge.svg?token=88P0HPY7G7"/></a>
</p>

<p align=center>
  <a href="TODO:link.to.paper.com"><img alt="Paper Badge" src="https://img.shields.io/badge/paper-dinkel-informational"></a>
</p>

<details>

<summary><b>Table of Contents</b></summary>

- [📖 Overview](#-overview)
- [⚙ Installation](#-installation)
- [🔎 Fuzzing with Dinkel](#-fuzzing-with-dinkel)
  - [Prometheus Exporter](#prometheus-exporter)
- [💻 Contributing](#-contributing)
- [🐛 Bugs found by Dinkel](#-bugs-found-by-dinkel)

</details>

</br>

# INFO FOR REVIEWER

### Bug reports

A list of all the bugs referenced in the paper can be found at the bottom of this README.

### System disclaimer

Dinkel was tested on amd64 systems running linux. It is possible that some features may not work on others, though the fuzzing process itself should be unaffected.

### Downloading dinkel

To download dinkel, refer to the releases and download an appropriate version.

### Testing dinkel

Before running dinkel, ensure that there is a `bugreports` directory present in your PWD, otherwise dinkel will not write out bugreports.

In order to quickly see the effectiveness of dinkel, we recommend running it against an older version of Neo4j, before dinkel was used to find bugs within Neo4j.
An easy way to do so is by spinning up a Neo4j 5.6.0 docker container, and running dinkel against it:

```
# Generate the config
./dinkel config

# Run neo4j
docker run --rm -it -p 7687:7687 \
  -e NEO4J_PLUGINS=\[\"apoc\"\] \
  -e NEO4J_AUTH=none \
  -e NEO4J_ACCEPT_LICENSE_AGREEMENT=yes \
  -e NEO4J_server+default_listen_address=0.0.0.0 \
	-e NEO4J_internal_cypher_parallel__runtime__support=all \
  neo4j:5.6.0

# Test neo4j
./dinkel fuzz neo4j
# Test neo4j for logic bugs
./dinkel fuzz neo4j 1
# Test neo4j and print all queries
./dinkel fuzz neo4j -v2
```

### Evaluation data

The data used for the evaluation section of the paper is in the `/data` directory:

| Subdirectory           | Description                                                                                                                                                                                                                                                      |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `error_bugs`           | Holds the data used to analyze different aspects of the bugs found, such as size, dependencies, target. Used to generate Figures 10 and 12 in the paper. The data is in the `.csv` file, while the data processing can be analyzed in the jupyter-notebook file. |
| `feature_distribution` | Holds the data used to analyze the distribution of Cypher features within the bugs, creates Figure 9 in the paper. The data is held within the `.csv` file, while the data processing happens within the jupyter-notebook file within the directory.             |
| `coverage`             | Holds the data and notebooks for the coverage testing runs.                                                                                                                                                                                                      |
| `runtime_testing`      | Holds the data used for the sensitivity analysis. Within the directory is a small bash script, which spins up a prometheus docker container, with which the data can be inspected. Simply run the bash script with the `-h` flag for help.                       |

---

# 📖 Overview

Dinkel is a state of the art Cypher fuzzer.\
It employs on-the-fly state manipulation and self-generating ASTs to generate complex and valid queries with countless data dependencies.

For more detailed information on how dinkel works and performs, please refer to its [paper](TODO:path.to.paper).

# ⚙ Installation

Requirements:

- Go 1.22.0 or higher

</br>

Install dinkel:

```
$ go install github.com/Anon10214/dinkel@latest
```

You should now be able to run dinkel from the command line using `dinkel`.\
If you encounter an error, ensure that the `GOBIN` environment variable is set and in your path.

</br>

Alternatively, you may clone this repository and build dinkel locally

```
$ git clone git@github.com:Anon10214/dinkel.git
$ cd dinkel
$ go build
```

You should now have a binary which you can run via `./dinkel`.

# 🔎 Fuzzing with Dinkel

⚠ Never run dinkel against a database holding data you don't want to lose, as it will get deleted. ⚠ ️

</br>

If you need more info about a certain command, run

```
dinkel help [command]
```

</br>

Ensure you have a config in your present working directory.  
If you cloned the repository, this config will already be in the project's root directory.  
Otherwise, you can generate the config by running

```
dinkel config
```

</br>

Before you start fuzzing a target, spin up an instance of said target. For this, you may want to use the already provided dockerfiles contained in this project's `dockerfiles` directory. To then fuzz the target, run

<pre>
dinkel fuzz <ins>target</ins> [strategy]
</pre>

You can list available targets and strategies using `dinkel help fuzz`.

</br>

Once a bug was found and a bug report got generated, run

```
dinkel reduce path/to/bugreport.yml
```

to reduce the generated query. Note that the reduction is not perfect and you might still have to further reduce the query manually.

</br>

To make sure dinkel doesn't report the same bug again, add a regex matching the error message to the targets config.  
The entry should be added to the list `<the target>.reportedErrors` in the config.

You can check that the regex correctly matches the error message by rerunning the bugreport and making sure dinkel now recognizes the query as a `REPORTED_BUG`.

</br>

If you found multiple bugs and thus have a lot of bugreports, you might find use in the command

```
dinkel bugreports
```

With this command you can easily rerun, regenerate, reduce, rename and delete your bug reports.

</br>

### Prometheus Exporter

If you wish to run the fuzzer for a prolonged time you might want to monitor its performance.\
You can do this by enabling the builtin prometheus exporter with the `--prometheus-exporter port` flag in the `fuzz` command.\
Setting this flag exposes the `/metrics` HTTP endpoint on the specified port, exposing prometheus metrics.
These metrics include:

1. query counts
1. statement counts
1. generation latencies
1. query latencies
1. count of query result types

# 💻 Contributing

Please don't hesitate to create issues if you find a bug in dinkel or wish to share an idea to improve the tool.\
Feel free to open a pull request if you have made any improvements to dinkel!

Please refer to [the contributing guidelines](CONTRIBUTING.md) for more information about how to contribute.

You might have an easier time getting started with developing dinkel after reading its [paper](link.to.paper).

# 🐛 Bugs found by Dinkel

If you find a bug using dinkel, remember to responsibly disclose it to the respective developers.\
Once the bug is fixed, you may send an E-Mail to [bugs@dinkel-fuzz.ch](mailto:bugs@dinkel-fuzz.ch?subject=[Bug]), containing a subject starting with "[Bug]".

### So far, dinkel has found over 127 bugs in three GDBMSs:

<details>

<summary>Neo4j</summary>

|   Type    | Link to GitHub Issue                        | Notes |
| :-------: | ------------------------------------------- | ----- |
| Exception | https://github.com/neo4j/neo4j/issues/13054 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13077 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13078 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13086 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13091 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13093 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13100 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13098 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13101 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13102 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13099 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13105 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13109 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13143 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13129 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13141 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13147 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13148 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13152 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13150 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13163 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13164 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13169 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13194 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13196 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13284 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13337 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13336 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13345 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13427 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13429 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13119 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13123 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13130 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13142 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13149 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13151 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13425 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13426 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13431 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13432 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13436 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13439 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13466 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13486 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13487 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13491 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13552 |       |
| Exception | https://github.com/neo4j/neo4j/issues/13568 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13605 |       |
|   Logic   | https://github.com/neo4j/neo4j/issues/13606 |       |

</details>

<details>

<summary>FalkorDB (Formerly known as RedisGraph)</summary>

|   Type    | Link to GitHub Issue                                 | Notes                    |
| :-------: | ---------------------------------------------------- | ------------------------ |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3041 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3042 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3043 |                          |
| Exception | https://github.com/RedisGraph/RedisGraph/issues/3044 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3051 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3052 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3058 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3059 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3060 |                          |
| Exception | https://github.com/RedisGraph/RedisGraph/issues/3061 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3062 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3063 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3064 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3065 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3072 |                          |
|   Crash   | https://github.com/RedisGraph/RedisGraph/issues/3073 |                          |
| Exception | https://github.com/RedisGraph/RedisGraph/issues/3104 |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/602      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/603      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/610      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/608      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/605      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/607      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/621      |                          |
|   Crash   | https://github.com/FalkorDB/FalkorDB/issues/622      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/624      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/627      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/629      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/625      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/626      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/637      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/651      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/650      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/694      |                          |
|   Crash   | Reported Confidentially                              | crash_1fcd0a9            |
|   Crash   | Reported Confidentially                              | crash_3607fe             |
|   Crash   | Reported Confidentially                              | crash_36398a             |
|   Crash   | Reported Confidentially                              | crash_datablock_getitem  |
|   Crash   | Reported Confidentially                              | crash_delete_record      |
|   Crash   | Reported Confidentially                              | crash_duplicate_entries  |
|   Crash   | Reported Confidentially                              | crash_getattributes      |
|   Crash   | Reported Confidentially                              | crash_invalid_function   |
|   Crash   | Reported Confidentially                              | db_hang_replacement_char |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/747      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/748      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/749      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/750      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/830      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/832      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/831      |                          |
| Exception | https://github.com/FalkorDB/FalkorDB/issues/829      |                          |
|   Crash   | Reported Confidentially                              | crash_rowIterator        |
|   Crash   | Reported Confidentially                              | crash_36882a             |
|   Crash   | Reported Confidentially                              | crash_record_gettype     |
|   Crash   | Reported Confidentially                              | paralyze_unicode         |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/996      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/997      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/998      |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/1018     |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/1030     |                          |
|   Logic   | https://github.com/FalkorDB/FalkorDB/issues/1031     |                          |
|   Crash   | Reported Confidentially                              | crash_371b29             |

</details>

<details>

<summary>Memgraph</summary>

|   Type    | Link to GitHub Issue                             | Notes                                                     |
| :-------: | ------------------------------------------------ | --------------------------------------------------------- |
|   Crash   | https://github.com/memgraph/memgraph/issues/887  |                                                           |
| Exception | https://github.com/memgraph/memgraph/issues/904  |                                                           |
|   Crash   |                                                  | crash_abort                                               |
|   Crash   |                                                  | crash_segfault_less                                       |
| Exception | https://github.com/memgraph/memgraph/issues/2089 |                                                           |
| Exception |                                                  | min_bool_bool - fixed independently before we reported it |
| Exception | https://github.com/memgraph/memgraph/issues/2090 |                                                           |
| Exception | https://github.com/memgraph/memgraph/issues/2091 |                                                           |
| Exception | https://github.com/memgraph/memgraph/issues/2092 |                                                           |
|   Logic   | https://github.com/memgraph/memgraph/issues/2094 |                                                           |
|   Logic   | https://github.com/memgraph/memgraph/issues/2093 |                                                           |
|   Crash   | https://github.com/memgraph/memgraph/issues/2841 |                                                           |
| Exception | https://github.com/memgraph/memgraph/issues/2842 |                                                           |
| Exception | https://github.com/memgraph/memgraph/issues/2843 |                                                           |

</details>
