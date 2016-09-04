package main

import (
	"io/ioutil"
	"log"
	"time"

	"poule/configuration"
	"poule/operations"
	"poule/operations/catalog"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var batchCommand = cli.Command{
	Name:   "batch",
	Usage:  "run groups of commands described in files",
	Action: doBatchCommand,
}

func doBatchCommand(c *cli.Context) {
	for _, arg := range c.Args() {
		b, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatalf("Failed to read file %q: %v", arg, err)
		}

		// Read the YAML configuration file identified by the argument.
		batchConfig := batchConfiguration{}
		if err := yaml.Unmarshal(b, &batchConfig); err != nil {
			log.Fatalf("Failed to read YAML file %q: %v", arg, err)
		}

		// Execute each command described as part of the YAML file.
		config := configuration.FromGlobalFlags(c)
		batchConfig.Override(config)
		for _, command := range batchConfig.Commands {
			descriptor, ok := catalog.ByNameIndex[command.Type]
			if !ok {
				log.Fatalf("Unknown operation %q in file %q", command.Type)
			}
			runSingleOperation(config, descriptor.OperationFromConfig(command.Settings))
		}
	}
}

type commandConfiguration struct {
	Type     string                   `yaml:"type"`
	Settings operations.Configuration `yaml:"settings"`
}

type batchConfiguration struct {
	Delay      *time.Duration         `yaml:"delay"`
	DryRun     *bool                  `yaml:"dry-run"`
	Repository *string                `yaml:"repository"`
	Token      *string                `yaml:"token"`
	TokenFile  *string                `yaml:"token-file"`
	Commands   []commandConfiguration `yaml:"commands"`
}

func (b *batchConfiguration) Override(c *configuration.Config) {
	if b.Delay != nil {
		c.Delay = *b.Delay
	}
	if b.DryRun != nil {
		c.DryRun = *b.DryRun
	}
	if b.Repository != nil {
		c.Repository = *b.Repository
	}
	if b.Token != nil {
		c.Token = *b.Token
	}
	if b.TokenFile != nil {
		c.TokenFile = *b.TokenFile
	}
}
