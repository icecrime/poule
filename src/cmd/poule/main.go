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
	return cli.Command{
		Category: "Operations",
		Flags:    descriptor.Flags(),
		Name:     descriptor.Name(),
		Usage:    descriptor.Description(),
		Action: func(c *cli.Context) {
			runSingleOperation(configuration.FromGlobalFlags(c), descriptor.OperationFromCli(c))
		},
	}
}

func runSingleOperation(c *configuration.Config, operation catalog.Operation) {
	if op, ok := operation.(operations.IssueOperation); ok {
		operations.RunIssueOperation(c, op)
	} else if op, ok := operation.(operations.PullRequestOperation); ok {
		operations.RunPullRequestOperation(c, op)
	} else {
		log.Fatalf("Invalid operation type: %#v", operation)
	}
}
