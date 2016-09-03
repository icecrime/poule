package main

import (
	"log"
	"os"
	"time"

	"poule/operations/catalog"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "poule"
	app.Usage = "Mass interact with GitHub issues & pull requests"
	app.Version = "0.2.0"

	for _, descriptor := range catalog.Index {
		app.Commands = append(app.Commands, descriptor.Command())
	}

	app.Flags = []cli.Flag{
		cli.DurationFlag{
			Name:  "delay",
			Usage: "Delay between GitHub operations",
			Value: 10 * time.Second,
		},
		cli.BoolTFlag{
			Name:  "dry-run",
			Usage: "Simulate operations",
		},
		cli.StringFlag{
			Name:  "repository",
			Usage: "GitHub repository",
		},
		cli.StringFlag{
			Name:  "token",
			Usage: "GitHub API token",
		},
		cli.StringFlag{
			Name:  "token-file",
			Usage: "GitHub API token file",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
