package main

import (
	"io/ioutil"
	"log"

	"poule/configuration"
	"poule/server"

	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

var serveCommand = cli.Command{
	Name:  "serve",
	Usage: "Operate as a daemon listening on GitHub webhooks",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "poule.toml",
			Usage: "Poule configuration",
		},
	},
	Action: doServeCommand,
}

func doServeCommand(c *cli.Context) {
	cfgPath := c.String("config")
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", cfgPath, err)
	}

	// Read the YAML configuration file identified by the argument.
	serveConfig := server.ServerConfiguration{}
	if err := yaml.Unmarshal(b, &serveConfig); err != nil {
		log.Fatalf("Failed to read config file %q: %v", cfgPath, err)
	}

	overrides := configuration.FromGlobalFlags(c)
	overrideConfig(&serveConfig.Config, overrides)

	s, err := server.NewServer(&serveConfig)
	if err != nil {
		log.Fatal(err)
	}

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

func overrideConfig(config, overrides *configuration.Config) {
	if !config.DryRun && overrides.DryRun {
		config.DryRun = overrides.DryRun
	}
	if overrides.Token != "" {
		config.Token = overrides.Token
	}
	if overrides.TokenFile != "" {
		config.TokenFile = overrides.TokenFile
	}
}
