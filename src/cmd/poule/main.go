package main

import (
	"fmt"
	"log"
	"os"

	"poule/configuration"
	"poule/operations"
	"poule/operations/catalog"
	"poule/operations/catalog/settings"
	"poule/utils"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "poule"
	app.Usage = "Mass interact with GitHub issues & pull requests"
	app.Version = "0.3.0"

	app.Commands = []cli.Command{
		batchCommand,
		serveCommand,
	}
	for i, _ := range catalog.Index {
		descriptor := catalog.Index[i]
		app.Commands = append(app.Commands, makeCommand(descriptor))
	}

	app.Flags = configuration.Flags()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func makeCommand(descriptor catalog.OperationDescriptor) cli.Command {
	clidesc := descriptor.CommandLineDescription()
	return cli.Command{
		Category:  "Operations",
		Flags:     append(clidesc.Flags, settings.FilteringFlag),
		Name:      clidesc.Name,
		Usage:     clidesc.Description,
		ArgsUsage: clidesc.ArgsUsage,
		Action: func(c *cli.Context) {
			if err := executeSingleOperation(c, descriptor); err != nil {
				fmt.Printf("FATAL: Executing single operation: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func executeSingleOperation(c *cli.Context, descriptor catalog.OperationDescriptor) error {
	config, err := configuration.FromGlobalFlags(c)
	if err != nil {
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
	return runSingleOperation(config, op, f)
}

func runSingleOperation(c *configuration.Config, op operations.Operation, filters []*utils.Filter) error {
	if filterIncludesIssues(filters) && op.Accepts()&operations.Issues == operations.Issues {
		if err := operations.Run(c, op, &operations.IssueRunner{}, filters); err != nil {
			return err
		}

	}
	if filterIncludesPullRequests(filters) && op.Accepts()&operations.PullRequests == operations.PullRequests {
		if err := operations.Run(c, op, &operations.PullRequestRunner{}, filters); err != nil {
			return err
		}
	}
	return nil
}

func filterIncludesIssues(filters []*utils.Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Impl.(utils.IsFilter); ok && f.PullRequestOnly {
			return false
		}
	}
	return true
}

func filterIncludesPullRequests(filters []*utils.Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Impl.(utils.IsFilter); ok && !f.PullRequestOnly {
			return false
		}
	}
	return true
}
