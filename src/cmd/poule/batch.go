package main

import "github.com/codegangsta/cli"

var batchCommand = cli.Command{
	Name:   "batch",
	Usage:  "run groups of commands described in files",
	Action: doBatchCommand,
}

func doBatchCommand(c *cli.Context) {
	for _, arg := range c.Args() {
	}
}

type batchConfiguration struct {
	global struct {
		dryrun     bool   `yaml:"dry-run"`
		repository string `yaml:"repository"`
		tokenfile  string `yaml:"token-file"`
	}
	commands []map[string]interface{}
}
