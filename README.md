Design:

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
