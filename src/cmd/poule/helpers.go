package main

import (
	"poule/configuration"
	"poule/operations/catalog"
	"poule/operations/settings"
	"poule/runner"

	"github.com/urfave/cli"
)

func executeSingleOperation(c *cli.Context, descriptor catalog.OperationDescriptor) error {
	config := configuration.FromGlobalFlags(c)
	if err := config.Validate(); err != nil {
		return err
	}
	f, err := settings.ParseCliFilters(c)
	if err != nil {
		return err
	}
	op, err := descriptor.OperationFromCli(c)
	if err != nil {
		return err
	}

	runner := runner.NewOperationRunner(config, op)
	runner.GlobalFilters = f
	return runner.HandleStock()
}
