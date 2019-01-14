package vault

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"github.com/hashicorp/vault/api"
	"github.com/justinbarrick/hone/pkg/logger"
)

type Vault struct {
	Address string `hcl:"address"`
	Token   string `hcl:"token"`
	client  *api.Client
}

func (v *Vault) Init() error {
	if v.Token != "" && v.Address != "" {
		logger.Printf("Initializing vault.")

		client, err := api.NewClient(&api.Config{
			Address: v.Address,
		})
		if err != nil {
			return err
		}

		v.client = client
		v.client.SetToken(v.Token)
	}

	return nil
}

func (v *Vault) LoadSecrets(workspace string, secrets []string) (map[string]string, error) {
	secretMap := map[string]string{}
	secretValuesMap := map[string]string{}
	secretPath := fmt.Sprintf("secret/data/hone/%s", workspace)

	if v.client != nil {
		c := v.client.Logical()

		secretValues, err := c.Read(secretPath)
		if err != nil {
			return nil, err
		}

		if secretValues != nil {
			valuesMap := secretValues.Data["data"].(map[string]interface{})
			for k, v := range valuesMap {
				secretValuesMap[k] = v.(string)
			}
		}
	}

	for _, secret := range secrets {
		secretSplit := strings.SplitN(secret, "=", 2)

		val := os.Getenv(secretSplit[0])
		if val == "" && len(secretSplit) > 1 {
			val = secretSplit[1]
		}

    secret := secretSplit[0]

		secretMap[secret] = val
		secretValue := secretValuesMap[secret]
		if secretValue != "" {
			secretMap[secret] = secretValue
		}
		if secretMap[secret] == "" {
			return nil, errors.New(fmt.Sprintf("Failed to load secret %s from vault or environment.", secret))
		}
	}

	if v.client != nil {
		c := v.client.Logical()

		_, err := c.Write(secretPath, map[string]interface{}{
			"data": secretMap,
		})
		if err != nil {
			return nil, err
		}
	}

	return secretMap, nil
}
