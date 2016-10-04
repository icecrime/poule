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

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
	"github.com/ehazlett/simplelog"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var batchCommand = cli.Command{
	Name:   "batch",
	Usage:  "Run groups of commands described in files",
	Action: doBatchCommand,
	Before: func(c *cli.Context) error {
		// set a simpler logrus formatter for better cli experience
		logrus.SetFormatter(&simplelog.SimpleFormatter{})
		return nil
	},
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

	// Read the configuration file identified by the argument.
	batchConfig := batchConfiguration{}
	if _, err := toml.Decode(string(b), &batchConfig); err != nil {
		return err
	}

	// Read the global configuration flags, and override them with the
	// specialized flags defined in the configuration file.
	config := configuration.FromGlobalFlags(c)
	batchConfig.applyConfig(config)

	// Execute each command described as part of the YAML file.
	for _, operationConfig := range batchConfig.Operations {
		logrus.Debugf("processing operation: %s", operationConfig.Type)
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
		if err := runSingleOperation(config, op, itemFilters); err != nil {
			logrus.Error(err)
		}
	}
	return nil
}

// we need a special type to allow toml to decode from a duration string
// see https://github.com/BurntSushi/toml#using-the-encodingtextunmarshaler-interface
type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type batchConfiguration struct {
	configuration.Config
	Operations []operationConfiguration `toml:"operations"`
}

type operationConfiguration struct {
	Type     string                   `toml:"type"`
	Filters  map[string]interface{}   `toml:"filters"`
	Settings operations.Configuration `toml:"settings"`
}

func (b *batchConfiguration) applyConfig(c *configuration.Config) {
	batchDelay := b.Delay()
	if batchDelay != time.Second*0 {
		c.SetDelay(batchDelay)
	}
	if !c.DryRun && b.DryRun {
		c.DryRun = b.DryRun
	}
	if c.Repository == "" {
		c.Repository = b.Repository
	}
	if c.Token == "" {
		c.Token = b.Token
	}
	if c.TokenFile == "" {
		c.TokenFile = b.TokenFile
	}
}
