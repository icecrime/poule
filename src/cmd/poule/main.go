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

	app.Commands = append(app.Commands, batchCommand)
	for i, _ := range catalog.Index {
		descrip := catalog.Index[i]
		command := cli.Command{
			Name:     descrip.Name(),
			Usage:    descrip.Description(),
			Category: "Single operations",
			Action: func(c *cli.Context) {
				runSingleOperation(
					configuration.FromGlobalFlags(c),
					descrip.OperationFromCli(c),
				)
			},
		}
		app.Commands = append(app.Commands, command)
	}

	app.Flags = configuration.Flags()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runSingleOperation(c *configuration.Config, operation catalog.Operation) {
	if op, ok := operation.(operations.IssueOperation); ok {
		operations.RunIssueOperation(c, op)
	} else if op, ok := operation.(operations.PullRequestOperation); ok {
		operations.RunPullRequestOperation(c, op)
	}
}
