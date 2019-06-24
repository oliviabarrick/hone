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

With no arguments, hone loads the configuration from `Honefile` in the local directory and the
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
* `outputs`: A list of files that this job outputs.
* `env`: A map of environment variables to add to the job.
* `engine`: An execution engine to use, defaults to docker or the global engine setting.
* `template`: the name of a Job template to use (see the section below on templates).
* `privileged`: if true, the container will be started in privileged mode.
* `service`: if true, the container will be started as a service, see [the section on Services](#Service).

When defining a job, a job's settings can be referenced in the context of another job:

```
job "build" {
    image = "golang:1.11.2"

    inputs = ["./cmd/", "./pkg/"]
    outputs = ["./bin"]

    shell = "go build ./bin ./cmd/"
}

job "release" {
    image = "alpine"

    inputs = jobs.build.outputs
    outputs = ["releases/"]

    shell = "mkdir releases/ && cp ${jobs.release.inputs[0]} releases/"
}
```

A reference to another job will create an implicit dependency between the two jobs. It is also
possible to reference settings from the current job, as seen above.

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

engine = ${env.ENGINE}
```

## Built-in variables

There are also some built-in variables:

* `env.GIT_TAG`: the tag present on this commit
* `env.GIT_BRANCH`: the branch that is checked out
* `env.GIT_COMMIT`: the current commit id.
* `env.GIT_COMMIT_SHORT`: an eight character short commit id.

## Conditions

It is possible to only run a job if it match certain conditions. Currently,
this has no effect on jobs that depend on the job with the condition, but the job
will only run when the condition is met.

Conditions are specified as a [YQL query](https://github.com/caibirdme/yql) using the
same `environ` context as other jobs.

For example, to only run a job when the branch is master and tagged, a job could be:

```
job "release" {
    image = "alpine"

    condition = "GIT_BRANCH='master' and GIT_TAG!=''"

    shell = "echo Release!"
}
```

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
are then exposed to the rest of the hone configuration in the `secrets` map and can be used
in most blocks, for example secrets can configure your cache:

```
# optional, namespace your vault secrets.
workspace = "${env.WORKSPACE}"

secrets = [
    "S3_ACCESS_KEY", "S3_SECRET_KEY"
]

env = [
    "VAULT_TOKEN", "WORKSPACE=dev"
]

vault {
    address = "http://127.0.0.1:8200"
    token = "${env.VAULT_TOKEN}"
}

cache {
    s3 {
        bucket = "mybucket"
        endpoint = "nyc3.digitaloceanspaces.com"
        access_key = "${secrets.S3_ACCESS_KEY}"
        secret_key = "${secrets.S3_SECRET_KEY}"
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

## Secrets format

Secrets can be specified as a name only, in which case they will be searched for in vault
or environment variables:

```
secrets = [ "S3_ACCESS_KEY" ]
```

You can also specify a default value for the secret if it is not set. This is useful for default
dev credentials or if a secret is optional (e.g., the s3 keys for cache):

```
secrets = ["S3_ENDPOINT=sfo2.digitaloceanspaces.com", "S3_ACCESS_KEY=", "S3_SECRET_KEY="]
```

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
        "DOCKER_USER" = "${env.DOCKER_USER}",
        "DOCKER_PASS" = "${env.DOCKER_PASS}",
    }

    shell = "kaniko --dockerfile=Dockerfile --context=/build/ --destination=${env.DOCKER_USER}/image:latest"
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
        "DOCKER_USER" = "${env.DOCKER_USER}",
        "DOCKER_PASS" = "${env.DOCKER_PASS}",
    }

    shell = "kaniko --dockerfile=Dockerfile --context=/build/ --destination=${env.DOCKER_USER}/image:latest"
}

# Default template
job "build" {
    shell = "go build -o binary ./cmd"
    outputs = ["binary"]
    inputs = ["./cmd/main.go"]
}

# Use the Docker template
job "docker-build" {
    template = "docker"
    inputs = ["binary"]
    deps = ["build"]
}
```

# Services

It is possible to create long running services that do not block jobs that depend on them.

These are created as services and take all of the same arguments as a job:

```
service "nginx" {
    image = "nginx:latest"
    exec = ["nginx", "-g", "daemon off;"]
}

job "curl" {
    deps = ["nginx"]
    image = "alpine"
    shell = "curl http://nginx/"
}
```

Nginx would be started, curl would run, and then nginx would be torn down at the end
of the build.

# Reporting to a Git repository

It is possible to report build status back to your Git provider. Hone has built in support
for a number of Git providers:

* Github
* Gitlab
* Gitea
* Gogs
* Bitbucket
* Stash

Your provider can be configured with a `repository` block, note that you can supply as many report blocks
as required:

```
repository {
    token = "github token"
}
```

The only required parameter is `token`, however, other options are:

* `token`: the token to use when authenticating with the provider, if it is empty, no status will be reported.
* `provider`: (optional) the provider to use, defaults to inferring from the configured remote or Github.
* `url`: (optional) the API URL for the provider, required for self-hosted providers.
* `repo`: (optional) the name of the repository to post status to, by default infers from the URL in the remote (or the `$REPO_OWNER` and `$REPO_NAME` variables).
* `remote`: (optional) the Git remote to attempt to infer provider configuration from, defaults to `origin`.
* `condition`: (optional) a condition that must be met in order to report the status. See the conditions section of the job for more information.
