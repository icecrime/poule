package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"poule/configuration"
	"poule/operations"
	"poule/operations/catalog"
	"poule/operations/catalog/settings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var batchCommand = cli.Command{
	Name:   "batch",
	Usage:  "Run groups of commands described in files",
	Action: doBatchCommand,
}

func doBatchCommand(c *cli.Context) {
	for _, arg := range c.Args() {
		if err := executeBatchFile(c, arg); err != nil {
			fmt.Printf("FATAL: Executing batch file %q: %v\n", arg, err)
			os.Exit(1)
		}
	}
}

func executeBatchFile(c *cli.Context, file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Read the YAML configuration file identified by the argument.
	batchConfig := batchConfiguration{}
	if err := yaml.Unmarshal(b, &batchConfig); err != nil {
		return err
	}

	// Read the global configuration flags, and override them with the
	// specialized flags defined in the YAML configuration file.
	config, err := configuration.FromGlobalFlags(c)
	if err != nil {
		return err
	}
	batchConfig.Override(config)

	// Execute each command described as part of the YAML file.
	for _, operationConfig := range batchConfig.Operations {
		descriptor, ok := catalog.ByNameIndex[operationConfig.Type]
		if !ok {
			return errors.Errorf("unknown operation %q in file %q", operationConfig.Type, file)
		}
		itemFilters, err := settings.ParseConfigurationFilters(operationConfig.Filters)
		if err != nil {
			return err
		}
		op, err := descriptor.OperationFromConfig(operationConfig.Settings)
		if err != nil {
			return err
		}
		runSingleOperation(config, op, itemFilters)
	}
	return nil
}

type batchConfiguration struct {
	Delay      *time.Duration           `yaml:"delay"`
	DryRun     *bool                    `yaml:"dry-run"`
	Repository *string                  `yaml:"repository"`
	Token      *string                  `yaml:"token"`
	TokenFile  *string                  `yaml:"token-file"`
	Operations []operationConfiguration `yaml:"operations"`
}

type operationConfiguration struct {
	Type     string                   `yaml:"type"`
	Filters  map[string]interface{}   `yaml:"filters"`
	Settings operations.Configuration `yaml:"settings"`
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
