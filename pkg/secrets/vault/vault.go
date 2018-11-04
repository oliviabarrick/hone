package vault

import (
	"github.com/hashicorp/vault/api"
	"github.com/davecgh/go-spew/spew"
	"fmt"
)

func LoadSecrets(workspace string, secrets []string) (map[string]string, error) {
	client, err := api.NewClient(&api.Config{
		Address: "http://172.17.0.2:8200",
	})
	if err != nil {
		return nil, err
	}

	client.SetToken("29QLdKrqiqSbgW3k0LO67kO9")

	secretMap := map[string]string{}

	for _, secret := range secrets {
		secretValue, err := client.Logical().Read(fmt.Sprintf("kv/hello"))
		if err != nil {
			return nil, err
		}
		spew.Dump(secretValue)
		secretMap[secret] = ""
	}

	return secretMap, nil
}
