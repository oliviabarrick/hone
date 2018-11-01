job "test" {
    image = "alpine"

    input = "myinput"
    output = "myoutput"

    shell = "go test ${build.input}"
}

job "curl" {
    image = "alpine"

    output = "google.html"
    shell = "curl https://google.com > google.html"

    deps = ["hello", "world"]
}

job "build" {
    deps = ["test", "curl"]

    image = "golang"

    env = {
        "GO111MODULE" = "on"
    }

    input = "./cmd/farm.go"
    output = "farm"

    shell = "go build -o ${build.output} ${build.input}"
}

job "hello" {
    image = "alpine"

    output = "hello"

    shell = "echo lol > ${hello.output}"
}

job "world" {
    image = "alpine"

    deps = ["hello"]

    outputs = {
        "mine" = "lol"
    }

    shell = <<EOF
echo world > ${world.outputs.mine}
cat ${hello.output} >> ${world.outputs.mine}
EOF
}

job "bye" {
    image = "alpine"

    output = "bye"

    shell = "echo bye > ${bye.output}"
}

job "moon" {
    image = "alpine"

    input = "${bye.output}"
    output = "moon"

    shell = "cat ${moon.input} > ${moon.output}"
}
