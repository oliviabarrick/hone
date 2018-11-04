variable "digitalocean_token" {}

job "packer_build" {
    image = "hashicorp/packer"

    input = "packer/template.json"

    outputs = {
        "version" = "VERSION"
    }

    shell =  <<EOF
packer build -machine-readable ${packer_build.input} |tee packer_output.log

grep digitalocean,artifact,0,id packer_output.log |cut -d: -f2 > ${packer_build.outputs.version}
EOF
}

job "terraform_test" {
    image = "golang"

    env = {
        "VERSION" = "${packer_build.output.version}"
    }

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "tests/kubernetes_test.go", "${packer_build.output.version}"
    ]

    shell = "VERSION=$(cat $VERSION) go test tests/kubernetes_test.go"
}

job "terraform_init" {
    image = "hashicorp/terraform"

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml"
    ]

    outputs = {
        "directory" = ".terraform"
    }

    shell = "terraform init -input=false terraform/"
}

job "terraform_deploy" {
    deps = ["terraform_test"]

    image = "hashicorp/terraform"

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "${terraform_init.output.directory}", "${packer_build.output.version}"
    ]

    env = {
        "VERSION" = "${packer_build.output.version}"
    }

    shell =  <<EOF
export TF_VAR_digitalocean_image=$(cat $VERSION)
terraform apply-input=false -auto-approve terraform/
EOF
}
