variable "digitalocean_token" {}

job "packer_build" "create" {
    image = "hashicorp/packer"

    input = [
        "packer/template.json"
    ]

    output = {
        "version": "VERSION"
    }

    shell =  <<EOF
packer build -machine-readable packer/template.json |tee packer_output.log

grep digitalocean,artifact,0,id packer_output.log |cut -d: -f2 > VERSION
EOF
}

job "packer_build" "delete" {
    image = "alpine/curl"

    output = {
        "version": "VERSION"
    }

    env = {
        "TOKEN": "${var.digitalocean_token}",
        "VERSION": "${packer_build.output.version}"
    }

    shell =  <<EOF
curl -X DELETE -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" "https://api.digitalocean.com/v2/images/$(cat $VERSION)"
EOF
}

job "terraform_test" {
    image = "golang"

    env = {
        "VERSION": "${packer_build.output.version}"
    }

    input = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "tests/kubernetes_test.go", "${packer_build.output.version}"
    ]

    shell = "VERSION=$(cat $VERSION) go test tests/kubernetes_test.go"
}

job "terraform_init" {
    image = "hashicorp/terraform"

    input = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml"
    ]

    output = {
        "directory": ".terraform"
    }

    shell = "terraform init -input=false terraform/"
}

job "terraform_deploy" "create" {
    deps = ["terraform_test"]

    image = "hashicorp/terraform"

    input = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "${terraform_init.output.directory}", "${packer_build.output.version}"
    ]

    env = {
        "VERSION": "${packer_build.output.version}"
    }

    shell =  <<EOF
export TF_VAR_digitalocean_image=$(cat $VERSION)
terraform apply-input=false -auto-approve terraform/
EOF
}

job "terraform_deploy" "delete" {
    image = "hashicorp/terraform"

    shell = "terraform destroy -input=false terraform/"
}
