package vault

import (
	"errors"
	"fmt"
	"github.com/hashicorp/vault/api"
)

type Vault struct {
	Address string `hcl:"address"`
	Token   string `hcl:"token"`
	client  *api.Client
}

func (v *Vault) Init() error {
	client, err := api.NewClient(&api.Config{
		Address: v.Address,
	})
	if err != nil {
		return err
	}

	v.client = client
	v.client.SetToken(v.Token)
	return nil
}

func (v *Vault) LoadSecrets(workspace string, secrets []string) (map[string]string, error) {
	secretMap := map[string]string{}
	secretPath := fmt.Sprintf("secret/data/farm/%s", workspace)
	c := v.client.Logical()

	secretValues, err := c.Read(secretPath)
	if err != nil {
		return nil, err
	}

	secretValuesMap := map[string]string{}
	if secretValues != nil {
		valuesMap := secretValues.Data["data"].(map[string]interface{})
		for k, v := range valuesMap {
			secretValuesMap[k] = v.(string)
		}
	}

	for _, secret := range secrets {
		secretMap[secret] = os.Getenv(secret)
		secretValue := secretValuesMap[secret]
		if secretValue != "" {
			secretMap[secret] = secretValue
		}
		if secretMap[secret] == "" {
			return nil, errors.New(fmt.Sprintf("Failed to load secret %s from vault or environment.", secret))
		}
	}

	_, err = c.Write(secretPath, map[string]interface{}{
		"data": secretMap,
	})
	if err != nil {
		return nil, err
	}

	return secretMap, nil
}
