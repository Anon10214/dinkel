# Thank You For Considering Contributing To Dinkel!


### Table of Contents:

- [📣 Creating an Issue](#-creating-an-issue)
- [🚀 Opening a Pull Request](#-opening-a-pull-request)
- [🛠 Dev Dependencies](#-dev-dependencies)
- [🎨 Style Guidelines](#-style-guidelines)
- [🧪 Testing](#-testing)
- [🎯 Adding a new Fuzzing Target](#-adding-a-new-fuzzing-target)
- [🏗 Project Structure](#-project-structure)

# 📣 Creating an Issue

If you found a bug in dinkel or wish to share a feature request, feel free to open an issue in this repo.

The provided issue templates depicts how they should be formatted and what info should be included.
Please make sure to follow their suggestions.

# 🚀 Opening a Pull Request

Before opening a pull request, please make sure all [tests](#-testing) pass after including your changes, and confirm that you followed the [style guidelines](#-style-guidelines).

Make sure to [mention](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue) relevant issues in the description if your pull request addresses them.

Currently, all pull requests have to be reviewed by **@Anon10214**.
You can rest assured that the requests will be reviewed, though you might have to be patient.

# 🛠 Dev Dependencies

Let's get to developing dinkel!

In order to make changes to dinkel's codebase, make sure you have installed the below dependencies.

- Go 1.20 or newer
- [middlewarer](https://github.com/Anon10214/middlewarer) for generating the **dbms.DB** and **scheduler.Strategy** middleware framework.\
  You might have to run `go generate ./...` in the project's root repository if you made changes to these types.

# 🎨 Style Guidelines

Pull requests will be blocked by our pipeline if `goftmt` or `go vet` find issues with your code.\
In order to avoid this, make sure you format your code and check it for common pitfalls using the aforementioned tools.

Dinkel does not yet adhere to rules imposed by `golint`, though we are planning on changing this.
Because of this, please try your best to ensure that `golint` doesn't complain about code you have pushed, in order to take off the burden of linting later on.

# 🧪 Testing

For running the basic tests, use

```
go test -v ./...
```

For integration testing, you need to be able to run [docker compose](https://docs.docker.com/compose/install) on your machine.\
Once you have installed docker compose, navigate to the project's root directory and spin up the relevant docker containers

```
docker compose -f integration/docker-compose.ci.yml up
```

These containers open ports on your local machine. If you encounter any errors, ensure no process is listening on the ports defined in the docker-compose file.

Now you can run the integration tests:

```
go test -v -tags=integration ./...
```

# 🎯 Adding a new Fuzzing Target

In order to add a new fuzzing target `<X>`, create a new directory `models/<X>` which holds a `driver.go` and `implementation.go` file.

Create a struct in the `driver.go` file named `Driver` and have it implement the `DB` interface defined in `dbms/dbms.go`.

Create a struct in the `implementation.go` file named `Implementation` and have it implement the `Implementation` interface defined in `translator/translator.go`.

In order for query regeneration (and thus, by extend, reduction) to work reliably, the driver's `GetSchema` method **has** to be deterministic.

Next, add references to the new target in the following places:

- `cmd/config/config.go:72`: Add a case statement for `"<X>"` and return the appropriate structs.
- `cmd/fuzz.go:26`: Add `<X>` to the help message listing all available targets.
- `cmd/fuzz.go:44`: Add `<X>` to the list of valid arguments.

Additionally, please create a corresponding dockerfile for your target in the `dockerfiles` directory.

You can now further improve the fuzzing of your new target by adjusting the drop-ins of its implementation and tweaking its OpenCypher config.

# 🏗 Project Structure

This section aims to give you an idea of how the project is set up and where to find relevant files.

```bash
├── translator
│   ├── helperclauses
│   │   └── ... # Simple clauses useful for generation
│   └── translator.go # Generates ASTs and translates them to queries
│                     # Also defines types needed for query generation
│
├── scheduler
│   ├── reduce.go # The reducer for query reduction
│   ├── scheduler.go # The scheduler glueing together all parts
│   └── ... # Fuzzing strategies
│
├── models
│   ├── opencypher
│   │   ├── rootclause.go # The OpenCypher root clause, the query AST's root
│   │   ├── clauses
│   │   │   └── ... # Clauses making up AST nodes
│   │   ├── config
│   │   │   └── config.go # Query generation config
│   │   └── schema
│   │       └── schema.go # The schema used for stateful query generation
│   │
│   ├── mock
│   │   └── ... # Mock model used for testing
│   │
│   └── <X>
│       └── ... # Model for fuzzing target <X>
│
├── dbms
│   └── dbms.go # Types defining a target driver
│
├── seed
│   ├── helpers.go # Helper functions for seeds
│   └── seed.go # Seeds used to guide fuzzing
│
├── cmd
│   └── ... # cobra-cli commands
│
├── dockerfiles
│   └── ... # Dockerfiles for fuzzing targets
│
├── integration
│   └── ... # Files for integration testing
│
├── Dockerfile # dinkel's Dockerfile
│
└── middleware # Middleware for implementations
   └── prometheus
       └── ... # Prometheus middleware, collecting and exposing fuzzing metrics
```