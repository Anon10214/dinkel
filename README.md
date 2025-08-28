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

### System disclaimer

Dinkel was tested on amd64 systems running linux. It is possible that some features may not work on others, though the fuzzing process itself should be unaffected.

### Downloading dinkel

To download dinkel, refer to the releases and download an appropriate version.

### Testing dinkel

Before running dinkel, ensure that there is a `bugreports` directory present in your PWD, otherwise dinkel will not write out bugreports.

In order to quickly see the effectiveness of dinkel, we recommend running it against an older version of Neo4j, before dinkel was used to find bugs within Neo4j.
An easy way to do so is by spinning up a Neo4j 5.6.0 docker container, and running dinkel against it:

```
docker run --rm -it -p 7687:7687 \
  -e NEO4J_PLUGINS=\[\"apoc\"\] \
  -e NEO4J_AUTH=none \
  -e NEO4J_ACCEPT_LICENSE_AGREEMENT=yes \
  -e NEO4J_server+default_listen_address=0.0.0.0 \
  neo4j:5.6.0

./dinkel fuzz neo4j
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
Once the bug is fixed, you may (TODO: maybe add a mailing list?).

### So far, dinkel has found over x bugs in four GDBMSs:

<details>

<summary>Neo4j</summary>

1. A
1. B
1. C

</details>

<details>

<summary>RedisGraph</summary>

1. A
1. B
1. C

</details>

<details>

<summary>FalkorDB</summary>

1. A
1. B
1. C

</details>

<details>

<summary>Apache AGE</summary>

1. A
1. B
1. C

</details>

<details>

<summary>Memgraph</summary>

1. A
1. B
1. C

</details>
