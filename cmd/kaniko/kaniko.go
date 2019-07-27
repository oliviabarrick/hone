package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/justinbarrick/hone/pkg/executors/local"
	"github.com/justinbarrick/hone/pkg/logger"
)

type DockerAuth struct {
	Auth string `json"auth"`
}

type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

func main() {
	logger.InitLogger(0, nil)

	if os.Getenv("DOCKER_USER") != "" && os.Getenv("DOCKER_PASS") != "" {
		config := DockerConfig{
			Auths: map[string]DockerAuth{},
		}

		auth := fmt.Sprintf("%s:%s", os.Getenv("DOCKER_USER"), os.Getenv("DOCKER_PASS"))
		token := base64.StdEncoding.EncodeToString([]byte(auth))

		registry := os.Getenv("DOCKER_REGISTRY")
		if registry == "" {
			registry = "index.docker.io"
		}

		os.Unsetenv("DOCKER_REGISTRY")
		os.Unsetenv("DOCKER_USER")
		os.Unsetenv("DOCKER_PASS")
		os.Setenv("DOCKER_CONFIG", "/kaniko/.docker/")

		config.Auths[fmt.Sprintf("https://%s/v1/", registry)] = DockerAuth{
			Auth: token,
		}

		err := os.MkdirAll("/kaniko/.docker", 0600)
		if err != nil {
			log.Fatal(err)
		}

		cfgFile, err := os.OpenFile("/kaniko/.docker/config.json", os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal(err)
		}

		err = json.NewEncoder(cfgFile).Encode(config)
		cfgFile.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	args := []string{"/executor"}
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	if err := local.Exec(args, local.ParseEnv(os.Environ())); err != nil {
		log.Fatal(err)
	}
}
