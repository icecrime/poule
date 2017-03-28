package main

import (
	"io/ioutil"
	"log"
	"strings"

	"poule/configuration"
	"poule/operations/catalog"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var validateCommand = cli.Command{
	Name:  "validate",
	Usage: "Validate a Poule repository configuration file",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "server-config",
			Value: "",
			Usage: "Poule server configuration",
		},
		cli.StringFlag{
			Name:  "repository-config",
			Value: "",
			Usage: "Poule repository configuration",
		},
	},
	Action: doValidateCommand,
}

func validateServerConfig(cfgPath string) (*configuration.Server, error) {
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read server configuration")
	}

	// Read the YAML configuration file identified by the argument.
	serveConfig := configuration.Server{}
	if err := yaml.Unmarshal(b, &serveConfig); err != nil {
		return nil, errors.Wrapf(err, "malformed server configuration %q", cfgPath)
	} else if errs := serveConfig.Validate(catalog.OperationValidator{}); len(errs) != 0 {
		var strErrors []string
		for _, err := range errs {
			strErrors = append(strErrors, err.Error())
		}
		return nil, errors.New(strings.Join(strErrors, "\n"))
	}

	return &serveConfig, nil
}

func validateRepositoryConfig(cfgPath string) ([]configuration.Action, error) {
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read repository configuration")
	}

	var repoConfig configuration.Actions
	if err := yaml.Unmarshal(b, &repoConfig); err != nil {
		return nil, errors.Wrapf(err, "malformed repository configuration %q", cfgPath)
	} else if errs := repoConfig.Validate(catalog.OperationValidator{}); len(errs) != 0 {
		var strErrors []string
		for _, err := range errs {
			strErrors = append(strErrors, err.Error())
		}
		return nil, errors.New(strings.Join(strErrors, "\n"))
	}

	return repoConfig, nil
}

func doValidateCommand(c *cli.Context) {
	serverCfgPath := c.String("server-config")
	repositoryCfgPath := c.String("repository-config")

	if serverCfgPath == "" && repositoryCfgPath == "" {
		log.Fatal("specify --server-config and/or --repository-config")
	}

	if serverCfgPath != "" {
		if _, err := validateServerConfig(serverCfgPath); err != nil {
			log.Fatalf("Invalid server configuration %q:\n%s\n", serverCfgPath, err)
		}
		log.Println("Server configuration file is valid")

	}
	if repositoryCfgPath != "" {
		if _, err := validateRepositoryConfig(repositoryCfgPath); err != nil {
			log.Fatalf("Invalid repository configuration %q:\n%s\n", repositoryCfgPath, err)
		}
		log.Println("Repository configuration file is valid")
	}
}
