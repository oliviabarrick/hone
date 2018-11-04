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
GO111MODULE=on go get -u github.com/justinbarrick/farm/cmd/farm
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

# Caching

By default it uses a local file cache. To override this and use S3 as a cache,
set:

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
loaded as environment variables and then written into Vault if they don't already exist. Secrets
are then exposed to the rest of the farm configuration as environment variables and can be used
in most blocks, for example secrets can configure your cache:

```
# optional, namespace your vault secrets.
workspace = "${environ.WORKSPACE}"

secrets = [
    "S3_ACCESS_KEY", "S3_SECRET_KEY"
]

env = [
    "S3_ACCESS_KEY", "S3_SECRET_KEY", "VAULT_TOKEN", "WORKSPACE=dev"
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

Take the root token from the vault logs and invoke farm:

```
export VAULT_TOKEN=$TOKEN
export S3_ACCESS_KEY="myaccesskey"
export S3_SECRET_KEY="mysecretkey"
farm secrets.hcl build
```

Future runs will only require the vault token:

```
export VAULT_TOKEN=$TOKEN
farm secrets.hcl build
```

Environments can optionally be defined via the `workspace` setting, which will use
a different namespace for storing secrets (allowing you to easily switch between a
development and prod namespace).
