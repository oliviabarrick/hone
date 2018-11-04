env = [
    "DIGITALOCEAN_TOKEN"
]

job "packer_build" {
    image = "hashicorp/packer"

    input = "packer/*"

    outputs = [
        "VERSION"
    ]

    shell =  <<EOF
packer build -machine-readable packer/template.json |tee packer_output.log

grep digitalocean,artifact,0,id packer_output.log |cut -d: -f2 > VERSION
EOF
}

job "terraform_test" {
    image = "golang"

    deps = [ "packer_build" ]

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "tests/kubernetes_test.go", "VERSION"
    ]

    shell = "VERSION=$(cat VERSION) go test tests/kubernetes_test.go"
}

job "terraform_init" {
    image = "hashicorp/terraform"

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml"
    ]

    shell = "terraform init -input=false terraform/"
}

job "terraform_deploy" {
    deps = ["terraform_test"]

    image = "hashicorp/terraform"

    inputs = [
        "terraform/cluster.tf", "terraform/kubernetes/retrieve-kubeconfig.sh",
        "terraform/kubernetes/*.tf", "terraform/kubernetes/addons/*.yaml",
        "VERSION"
    ]

    shell =  <<EOF
export TF_VAR_digitalocean_image=$(cat VERSION)
terraform apply-input=false -auto-approve terraform/
EOF
}
