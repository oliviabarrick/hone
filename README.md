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

# Caching

By default it uses a local file cache. To override this and use S3 as a cache,
set:

* `S3_BUCKET`: the bucket to use.
* `S3_URL`: your S3 URL.
* `S3_ACCESS_KEY`: your access token.
* `S3_SECRET_KEY`: your secret token.
