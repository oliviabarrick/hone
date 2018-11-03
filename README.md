# Design

* Remote state, farm calculates a desired build state by creating a DAG of build resources
  using the paths and hashes of input files. This state is stored in a remote store so that
  it can be compared against the local filesystem wherever it is ran.
* A single build step is called a "job" and will be provided with its input files and
  build commands, it will return the output files and hashes.
* Build artifacts and input files are stored in S3-compatible data stores.
* Jobs can be run on any node and are launched in dependency order based on the DAG.
* Jobs will be run incrementally, so will only be run when input files have been changed.
* Builds will be configured using HCL.
* Jobs are configured as makefile-like targets.
* Dependencies can be defined implicitly or explicitly.
* CRUD handlers are easy to implement.

# Building

```
GO111MODULE=on go get -u github.com/justinbarrick/farm
```

# Running

```
farm examples/helloworld.tf build
```

# Job specification

You can have as many jobs as necessary. Each job requires a name, docker image, and shell.

You can also specify other jobs that it depends on, environment variables, input files,
and output files for caching.

```
job "test" {
    image = "golang:1.11.2"

    inputs = ["./cmd/", "./pkg/"]

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    shell = "go test ./cmd/... ./pkg/..."
}
```

Settings:

* `image`: The image to use.
* `shell`: Shell commands to run.
* `deps`: A list of jobs that this job depends on.
* `inputs`: A list of files, directories, or globs that this job depends on.
* `input`: A file, directory, or glob that this job depends on.
* `outputs`: A list of files that this job outputs.
* `output`: A file that this job outputs.
* `env`: A map of environment variables to add to the job.

# Execution engine

The tool can use Docker or Kubernetes as a backend for executing builds.

Set `engine = "docker"` to use Docker or `engine = "kubernetes"` to use Kubernetes.

# Environment variables

You can pass in environment variables to use in your configuration in the `env` key:

```
env = ["VAR1", "VAR2=abc"]
```

Environment variables are optional, they default to an empty string unless the environment
variable contains an equals sign in which case the data on the right of the `=` will be the
default value.

Environment variables can be referenced anywhere else in the configuration:

```
env = [
    "ENGINE=docker"
]

engine = ${environ.ENGINE}
```

# Caching

By default it uses a local file cache. To override this and use S3 as a cache,
set:

```
cache {
    s3 {
        # the bucket to use
        "bucket" = "mybucket"
        # s3 endpoint
        "endpoint" = "nyc3.digitaloceanspaces.com"
        # s3 access key
        "access_key" = "blah"
        # s3 access key
        "secret_key" = "blah"
    }
}
```

You can also override the file path for the file cache:

```
cache {
    file {
        cache_dir = "/my/cache/directory"
    }
}
```
