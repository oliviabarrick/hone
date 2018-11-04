env = [
    "ENGINE=docker",
    "S3_BUCKET=farm-cache-bucket",
    "S3_ENDPOINT=nyc3.digitaloceanspaces.com",
    "S3_ACCESS_KEY",
    "S3_SECRET_KEY",
    "S3_ENABLED"
]

engine = "${environ.ENGINE}"

cache {
    s3 {
        access_key = "${environ.S3_ACCESS_KEY}"
        secret_key = "${environ.S3_SECRET_KEY}"
        endpoint = "${environ.S3_ENDPOINT}"
        bucket = "${environ.S3_BUCKET}"
        disabled = "${environ.S3_ENABLED != "true"}"
    }
}

job "all" {
    deps = ["k8s-farm"]
    image = "alpine"
    shell = "echo all"
}

job "test" {
    image = "golang:1.11.2"

    inputs = ["./cmd/", "./pkg/", "go.mod", "go.sum"]

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    shell = "go test ./cmd/... ./pkg/..."
}

job "build-cache-shim" {
    image = "golang:1.11.2"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
        "CGO_ENABLED" = "0"
    }

    inputs = ["./cmd/*/*.go", "./pkg/**/*.go", "go.mod", "go.sum"]
    output = "./docker/cache-shim"

    shell = "go build -ldflags '-w -extldflags -static' -o ./docker/cache-shim ./cmd/cache-shim"
}

job "build" {
    deps = ["test"]

    image = "golang:1.11.2"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    inputs = ["./cmd/*/*.go", "./pkg/**/*.go", "go.mod", "go.sum"]
    output = "farm"

    shell = "go build -v -o ./farm ./cmd/farm"
}

job "build-mac" {
    deps = ["test"]

    image = "golang:1.11.2"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocachedarwin"
        "GOPATH" = "/build/.go"
        "GOOS" = "darwin"
    }

    inputs = ["./cmd/*/*.go", "./pkg/**/*.go", "go.mod", "go.sum"]
    output = "farm_darwin"

    shell = "go build -v -o ./farm_darwin ./cmd/farm"
}

job "k8s-farm" {
    image = "golang:1.11.2"

    env = {
        "KUBECONFIG" = "/build/.kubeconfig"
        "ENGINE" = "kubernetes"
        "S3_ACCESS_KEY" = "${environ.S3_ACCESS_KEY}"
        "S3_SECRET_KEY" = "${environ.S3_SECRET_KEY}"
        "S3_ENDPOINT" = "${environ.S3_ENDPOINT}"
        "S3_BUCKET" = "${environ.S3_BUCKET}"
        "S3_ENABLED" = "true"
    }

    deps = [ "build" ]

    shell = "./farm examples/helloworld.tf build"
}

job "hello" {
    image = "debian:stretch"

    output = "hello"

    shell = "echo hilol > hello"
}

job "world" {
    image = "debian:stretch"

    deps = ["hello"]

    outputs = [
        "lol"
    ]

    inputs = ["hello"]

    shell = <<EOF
cat hello > lol
EOF
}
