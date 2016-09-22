package main

import (
	"io/ioutil"
	"log"

	"poule/server"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
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
		cli.StringFlag{
			Name:  "listen, l",
			Value: ":8080",
			Usage: "Address on which to listen",
		},
		cli.StringFlag{
			Name:  "nsq-lookupd, n",
			Value: "127.0.0.1:4161",
			Usage: "Address of NSQ lookupd",
		},
	},
	Action: doServeCommand,
}

func doServeCommand(c *cli.Context) {
	// TODO
	// - Read mandatory configuration file associating event types (e.g., pull
	//   request created) with a set of operations
	// - Listen to GitHub webhooks with NSQ
	// - Match event types and execute associated operations
	cfgPath := c.String("config")
	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", cfgPath, err)
	}

	// Read the YAML configuration file identified by the argument.
	serveConfig := server.ServerConfig{}
	if _, err := toml.Decode(string(b), &serveConfig); err != nil {
		log.Fatalf("Failed to read config file %q: %v", cfgPath, err)
	}

	serveConfig.ListenAddr = c.String("listen")
	serveConfig.NSQLookupdAddr = c.String("nsq-lookupd")

	s, err := server.NewServer(&serveConfig)
	if err != nil {
		log.Fatal(err)
	}

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
