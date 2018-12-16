A build tool designed to make quality CI configurations and incremental builds easy to achieve. It
can replace Make, your CI server, and seamlessly orchestrate workloads running locally in Docker or
Kubernetes.

# Design

* Remote state, hone calculates a desired build state by creating a DAG of build resources
  using the paths and hashes of input files. This state is stored in a remote store so that
  it can be compared against the local filesystem wherever it is ran.
* A single build step is called a "job" and will be provided with its input files and
  build commands, it will return the output files and hashes.
* Build artifacts and input files are stored in S3-compatible data stores.
* Jobs can be run in Kubernetes or Docker (or any other supported execution engine) and are launched in dependency order based on the DAG.
* Jobs will be run incrementally, so will only be run when input files have been changed.
* Builds will be configured using HCL.
* Jobs are configured as makefile-like targets.
* Easy integration with Vault for secrets management.

# Building

```
GO111MODULE=on go get -u github.com/justinbarrick/hone/cmd/hone
```

# Running

```
hone examples/helloworld.hcl build
```

# Configuration

With no argumentsn, hone loads the configuration from `Honefile` in the local directory and the
`all` target.

A single argument to hone specifies which target to use:

```
hone build
```

You can specify an alternative Honefile as well:

```
hone examples/helloworld.hcl build
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
* `exec`: A command to execute without using bash (as a string array).
* `deps`: A list of jobs that this job depends on.
* `inputs`: A list of files, directories, or globs that this job depends on.
* `input`: A file, directory, or glob that this job depends on.
* `outputs`: A list of files that this job outputs.
* `output`: A file that this job outputs.
* `env`: A map of environment variables to add to the job.
* `engine`: An execution engine to use, defaults to docker or the global engine setting.
* `template`: the name of a Job template to use (see the section below on templates).

# Execution engine

The tool can use the host, Docker or Kubernetes as a backend for executing builds.

Available engines:

* `docker`: the default, executes in Docker containers.
* `kubernetes`: run containers in Kubernetes instead of local Docker.
* `local`: run commands directly on the host without using containers.

You can set the engine globally or on the job (the job's engine setting overrides the global engine setting).

Currently using Kubernetes requires using the S3 cache backend. The Kubernetes namespace and configuration
file are configurable via the `kubernetes` block:

```
kubernetes {
    namespace = "default"
    kubeconfig = "/path/to/kubeconfig"
}
```

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

## Built-in variables

There are also some built-in variables:

* `environ.GIT_TAG`: the tag present on this commit
* `environ.GIT_BRANCH`: the branch that is checked out
* `environ.GIT_COMMIT`: the current commit id.
* `environ.GIT_COMMIT_SHORT`: an eight character short commit id.

# Caching

By default it uses a local file cache. To also use S3 as a cache, set:

```
cache {
    s3 {
        # the bucket to use
        bucket = "mybucket"
        # s3 endpoint
        endpoint = "nyc3.digitaloceanspaces.com"
        # s3 access key
        access_key = "blah"
        # s3 access key
        secret_key = "blah"
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

# Secrets management with Vault

Secrets can be stored in Vault instead of being passed as environment variables. Secrets are first
loaded from environment variables and then written into Vault if they don't already exist. Secrets
are then exposed to the rest of the hone configuration as environment variables and can be used
in most blocks, for example secrets can configure your cache:

```
# optional, namespace your vault secrets.
workspace = "${environ.WORKSPACE}"

secrets = [
    "S3_ACCESS_KEY", "S3_SECRET_KEY"
]

env = [
    "VAULT_TOKEN", "WORKSPACE=dev"
]

vault {
    address = "http://127.0.0.1:8200"
    token = "${environ.VAULT_TOKEN}"
}

cache {
    s3 {
        bucket = "mybucket"
        endpoint = "nyc3.digitaloceanspaces.com"
        access_key = "${environ.S3_ACCESS_KEY}"
        secret_key = "${environ.S3_SECRET_KEY}"
    }
}

job "build" {
    image = "alpine"

    shell = "uname -a"
}
```

Save as `secrets.hcl`.

Start a vault instance:

```
docker run --cap-add=IPC_LOCK -p 8200 --name=dev-vault vault
```

Take the root token from the vault logs and invoke hone:

```
export VAULT_TOKEN=$TOKEN
export S3_ACCESS_KEY="myaccesskey"
export S3_SECRET_KEY="mysecretkey"
hone secrets.hcl build
```

Future runs will only require the vault token:

```
export VAULT_TOKEN=$TOKEN
hone secrets.hcl build
```

Environments can optionally be defined via the `workspace` setting, which will use
a different namespace for storing secrets (allowing you to easily switch between a
development and prod namespace).

# Building Docker images

Kaniko is recommended for building Docker images, a custom Kaniko shim has been built
to write a Docker configuration using a password and username loaded from the environment.

To use:

```
env = [
    "DOCKER_USER", "DOCKER_PASS"
]

job "docker-build" {
    image = "justinbarrick/kaniko:latest"

    env = {
        "DOCKER_USER" = "${environ.DOCKER_USER}",
        "DOCKER_PASS" = "${environ.DOCKER_PASS}",
    }

    shell = "kaniko --dockerfile=Dockerfile --context=/build/ --destination=${environ.DOCKER_USER}/image:latest"
}
```

You can also set `DOCKER_REGISTRY` to use a different Docker registry.

# Templates

Hone supports job templates to reduce duplication of similar jobs. Templates can have
any name, however, all jobs will use the `default` template if it exists if a template
is not supplied.

```
env = [
    "DOCKER_USER", "DOCKER_PASS"
]

template "default" {
    image = "golang:1.11.2"
}

template "docker" {
    image = "justinbarrick/kaniko:latest"

    env = {
        "DOCKER_USER" = "${environ.DOCKER_USER}",
        "DOCKER_PASS" = "${environ.DOCKER_PASS}",
    }

    shell = "kaniko --dockerfile=Dockerfile --context=/build/ --destination=${environ.DOCKER_USER}/image:latest"
}

# Default template
job "build" {
    shell = "go build -o binary ./cmd"
    output = "binary"
    input = "./cmd/main.go"
}

# Use the Docker template
job "docker-build" {
    template = "docker"
    input = "binary"
    deps = ["build"]
}
