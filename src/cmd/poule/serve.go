package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var serveCommand = cli.Command{
	Name:   "serve",
	Usage:  "Operate as a daemon listening on GitHub webhooks",
	Action: doServeCommand,
}

func doServeCommand(c *cli.Context) {
	// TODO
	// - Read mandatory configuration file associating event types (e.g., pull
	//   request created) with a set of operations
	// - Listen to GitHub webhooks with NSQ
	// - Match event types and execute associated operations
	b, err := ioutil.ReadFile(c.Args()[0])
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", c.Args()[0], err)
	}

	// Read the YAML configuration file identified by the argument.
	serveConfig := serveConfiguration{}
	if err := yaml.Unmarshal(b, &serveConfig); err != nil {
		log.Fatalf("Failed to read YAML file %q: %v", c.Args()[0], err)
	}

	fmt.Printf("%#v\n", serveConfig.Events)
}

type serveConfiguration struct {
	Delay      *time.Duration       `yaml:"delay"`
	DryRun     *bool                `yaml:"dry-run"`
	Repository *string              `yaml:"repository"`
	Token      *string              `yaml:"token"`
	TokenFile  *string              `yaml:"token-file"`
	Events     []eventConfiguration `yaml:"events"`
}

type eventConfiguration struct {
	Type       string                   `yaml:"type"`
	Action     string                   `yaml:"action"`
	Operations []operationConfiguration `yaml:"operations"`
}
