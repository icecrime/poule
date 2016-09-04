package configuration

import (
	"time"

	"github.com/urfave/cli"
)

type Config struct {
	Delay      time.Duration `yaml:"delay"`
	DryRun     bool          `yaml:"dry-run"`
	Repository string        `yaml:"repository"`
	Token      string        `yaml:"token"`
	TokenFile  string        `yaml:"token-file"`
}

func Flags() []cli.Flag {
	return []cli.Flag{
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
}

func FromGlobalFlags(c *cli.Context) *Config {
	return &Config{
		Delay:      c.GlobalDuration("delay"),
		DryRun:     c.GlobalBool("dry-run"),
		Repository: c.GlobalString("repository"),
		Token:      c.GlobalString("token"),
		TokenFile:  c.GlobalString("token-file"),
	}
}
