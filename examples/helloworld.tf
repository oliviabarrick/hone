job "test" {
    image = "golang"

    inputs = ["./cmd/", "./pkg/"]

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    shell = "go test ./cmd/... ./pkg/..."
}

job "build" {
    deps = ["test", "curl"]

    image = "golang"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    inputs = ["./cmd/*", "./pkg/*/*"]
    output = "farm"

    shell = "go build -v -o ./farm ./cmd/farm.go"
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

    shell = "echo lol"
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

job "bye" {
    image = "alpine"

    output = "bye"

    shell = "echo bye > bye"
}

job "moon" {
    image = "alpine"

    input = "bye"
    output = "moon"

    shell = "cat bye > moon"
}
