package main

import (
	"log"
	"os"

	"cmd/poule/commands"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "poule"
	app.Usage = "Mass interact with GitHub pull requests"
	app.Version = "0.1.0"

	app.Commands = []cli.Command{
		commands.AuditCommand,
		commands.CleanCommand,
		commands.RebuildCommand,
	}

	app.Flags = []cli.Flag{
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
