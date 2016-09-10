package main

import (
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
		Category: "Operations",
		Flags:    append(clidesc.Flags, settings.FilteringFlag),
		Name:     clidesc.Name,
		Usage:    clidesc.Description,
		Action: func(c *cli.Context) {
			f, err := settings.ParseCliFilters(c)
			if err != nil {
				log.Fatalf("Error parsing CLI: %v", err)
			}
			runSingleOperation(configuration.FromGlobalFlags(c), descriptor.OperationFromCli(c), f)
		},
	}
}

func runSingleOperation(c *configuration.Config, op operations.Operation, filters []*utils.Filter) {
	if filterIncludesIssues(filters) && op.Accepts()&operations.Issues == operations.Issues {
		operations.RunOnIssues(c, op, filters)
	}
	if filterIncludesPullRequests(filters) && op.Accepts()&operations.PullRequests == operations.PullRequests {
		operations.RunOnPullRequests(c, op, filters)
	}
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
