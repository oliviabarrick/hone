job "test" {
    image = "golang:1.11.2"

    env = {
        "GO111MODULE" = "on"
        "GOCACHE" = "/build/.gocache"
        "GOPATH" = "/build/.go"
    }

    inputs = ["./cmd/*/*.go", "./pkg/**/*.go", "go.mod", "go.sum"]

    shell = "go test ./cmd/... ./pkg/..."
}

job "build" {
    deps = [ "test" ]

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
