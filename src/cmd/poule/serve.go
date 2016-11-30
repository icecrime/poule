package main

import (
	"log"

	"poule/configuration"
	"poule/server"

	"github.com/urfave/cli"
)

var serveCommand = cli.Command{
	Name:  "serve",
	Usage: "Operate as a daemon listening on GitHub webhooks",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "poule-server.yml",
			Usage: "Poule server configuration",
		},
	},
	Action: doServeCommand,
}

func doServeCommand(c *cli.Context) {
	serveConfig, err := validateServerConfig(c.String("config"))
	if err != nil {
		log.Fatal(err)
	}
	overrides := configuration.FromGlobalFlags(c)
	overrideConfig(&serveConfig.Config, overrides)

	// Create the server.
	s, err := server.NewServer(serveConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize repositories specific configuration from GitHub.
	if err := s.FetchRepositoriesConfigs(); err != nil {
		log.Fatal(err)
	}

	// Start the long-running job.
	s.Run()
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
