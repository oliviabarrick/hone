env = [
    "ENGINE=docker",
    "BUCKET=farm-cache-bucket",
    "ACCESS_KEY",
    "SECRET_KEY"
]

engine = "${environ.ENGINE}"

cache {
    s3 {
        access_key = "${environ.ACCESS_KEY}"
        secret_key = "${environ.SECRET_KEY}"
        endpoint = "nyc3.digitaloceanspaces.com"
        bucket = "${environ.BUCKET}"
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

    shell = "kubectl delete pod test build || :"
}

job "test" {
    image = "golang:1.11"

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

    image = "golang:1.11"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    inputs = ["./cmd/*.go", "./pkg/**/*.go"]
    output = "farm"

    shell = "go build -v -o ./farm ./cmd/farm.go"
}

job "build-mac" {
    deps = ["test", "build"]

    image = "golang:1.11.1"

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
        "USE_KUBERNETES" = "1"
    }

    input = "./farm"
    deps = [ "build", "cleanup" ]

    shell = "./farm examples/helloworld.tf curl"
}

job "curl" {
    image = "byrnedo/alpine-curl"

    output = "google.html"
    shell = "curl https://google.com > google.html"

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
