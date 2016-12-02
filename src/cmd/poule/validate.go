package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"poule/configuration"

	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var validateCommand = cli.Command{
	Name:  "validate",
	Usage: "Validate a Poule repository configuration file",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "server-config",
			Value: "poule-server.yml",
			Usage: "Poule server configuration",
		},
		cli.StringFlag{
			Name:  "repository-config",
			Value: "poule.yml",
			Usage: "Poule repository configuration",
		},
	},
	Action: doValidateCommand,
}

func validateServerConfig(cfgPath string) (*configuration.Server, error) {
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %q: %v", cfgPath, err)
	}

	// Read the YAML configuration file identified by the argument.
	serveConfig := configuration.Server{}
	if err := yaml.Unmarshal(b, &serveConfig); err != nil {
		return nil, fmt.Errorf("Failed to read config file %q: %v", cfgPath, err)
	}

	return &serveConfig, nil
}

func validateRepositoryConfig(cfgPath string) ([]configuration.Action, error) {
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %q: %v", cfgPath, err)
	}

	var repoConfig []configuration.Action
	if err := yaml.Unmarshal(b, &repoConfig); err != nil {
		return nil, fmt.Errorf("failed to read repository configuration file: %v", err)
	}

	return repoConfig, nil
}

func doValidateCommand(c *cli.Context) {
	if cfgPath := c.String("server-config"); cfgPath != "" {
		if _, err := validateServerConfig(cfgPath); err != nil {
			log.Fatal(err)
		}
		log.Println("server configuration file is valid")

	}
	if cfgPath := c.String("repository-config"); cfgPath != "" {
		if _, err := validateRepositoryConfig(cfgPath); err != nil {
			log.Fatal(err)
		}
		log.Println("repository configuration file is valid")
	}
}
