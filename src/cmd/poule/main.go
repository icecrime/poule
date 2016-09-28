package main

import (
	"fmt"
	"log"
	"os"

	"poule/configuration"
	"poule/operations/catalog"
	"poule/operations/catalog/settings"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "poule"
	app.Usage = "Mass interact with GitHub issues & pull requests"
	app.Version = "0.3.0"
	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		return nil
	}

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
