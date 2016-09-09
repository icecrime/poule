package main

import (
	"log"
	"os"

	"poule/configuration"
	"poule/operations"
	"poule/operations/catalog"

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
	cliDescription := descriptor.CommandLineDescription()
	return cli.Command{
		Category: "Operations",
		Flags:    cliDescription.Flags,
		Name:     cliDescription.Name,
		Usage:    cliDescription.Description,
		Action: func(c *cli.Context) {
			runSingleOperation(configuration.FromGlobalFlags(c), descriptor.OperationFromCli(c))
		},
	}
}

func runSingleOperation(c *configuration.Config, op operations.Operation) {
	if op.Accepts()&operations.Issues == operations.Issues {
		operations.RunOnIssues(c, op)
	}
	if op.Accepts()&operations.PullRequests == operations.PullRequests {
		operations.RunOnPullRequests(c, op)
	}
}
