env = [
    "ENGINE=docker",
    "S3_BUCKET=farm-cache-bucket",
    "S3_ENDPOINT=nyc3.digitaloceanspaces.com",
    "S3_ACCESS_KEY",
    "S3_SECRET_KEY"
]

engine = "${environ.ENGINE}"

cache {
    s3 {
        access_key = "${environ.S3_ACCESS_KEY}"
        secret_key = "${environ.S3_SECRET_KEY}"
        endpoint = "${environ.S3_ENDPOINT}"
        bucket = "${environ.S3_BUCKET}"
    }
}

job "all" {
    deps = ["k8s-farm", "build-mac"]
    image = "alpine"
    shell = "echo all"
}

job "cleanup" {
    image = "lachlanevenson/k8s-kubectl"

    env = {
        "KUBECONFIG" = "/build/.kubeconfig"
    }

    shell = "kubectl delete pod hello world uname || :"
}

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

job "build" {
    deps = ["test"]

    image = "golang:1.11.2"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    inputs = ["./cmd/*.go", "./pkg/**/*.go"]
    output = "farm"

    shell = "go build -v -ldflags '-linkmode external -extldflags -static' -o ./farm ./cmd/farm.go"
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

    inputs = ["./cmd/*.go", "./pkg/**/*.go"]
    output = "farm_darwin"

    shell = "go build -v -o ./farm_darwin ./cmd/farm.go"
}

job "k8s-farm" {
    image = "lachlanevenson/k8s-kubectl"

    env = {
        "KUBECONFIG" = "/build/.kubeconfig"
        "ENGINE" = "kubernetes"
        "S3_ACCESS_KEY" = "${environ.S3_ACCESS_KEY}"
        "S3_SECRET_KEY" = "${environ.S3_SECRET_KEY}"
        "S3_ENDPOINT" = "${environ.S3_ENDPOINT}"
        "S3_BUCKET" = "${environ.S3_BUCKET}"
    }

    deps = [ "build", "cleanup" ]

    shell = "./farm examples/helloworld.tf uname"
}

job "uname" {
    image = "alpine"

    shell = "uname -a"

    deps = ["hello", "world"]
}

job "hello" {
    image = "alpine"

    output = "hello"

    shell = "echo lol > hello"
}

job "world" {
    image = "alpine"

    deps = ["hello"]

    outputs = [
        "lol"
    ]

    shell = <<EOF
echo world > lol
echo hi >> lol
EOF
}
