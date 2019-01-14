env = [
    "LOL=LOL"
]

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
    outputs = ["hone"]

    shell = "go build -v -o ./hone ./cmd/hone"
}
