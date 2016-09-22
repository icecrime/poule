package configuration

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type Config struct {
	Delay      time.Duration `yaml:"delay"`
	DryRun     bool          `yaml:"dry-run" toml:"dry_run"`
	Repository string        `yaml:"repository"`
	Token      string        `yaml:"token"`
	TokenFile  string        `yaml:"token-file" toml:"token_file"`
}

func (c *Config) SplitRepository() (string, string) {
	username, repository, err := getRepository(c.Repository)
	if err != nil {
		panic("invalid repository")
	}
	return username, repository
}

func (c *Config) Validate() error {
	if _, _, err := getRepository(c.Repository); err != nil {
		return err
	}
	return nil
}

func Flags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "enable debug logging",
		},
		cli.DurationFlag{
			Name:  "delay",
			Usage: "delay between GitHub operations",
			Value: 10 * time.Second,
		},
		cli.BoolTFlag{
			Name:  "dry-run",
			Usage: "simulate operations",
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

func FromGlobalFlags(c *cli.Context) (*Config, error) {
	config := &Config{
		Delay:      c.GlobalDuration("delay"),
		DryRun:     c.GlobalBool("dry-run"),
		Repository: c.GlobalString("repository"),
		Token:      c.GlobalString("token"),
		TokenFile:  c.GlobalString("token-file"),
	}
	return config, config.Validate()
}

func getRepository(repository string) (string, string, error) {
	s := strings.SplitN(repository, "/", 2)
	if len(s) != 2 {
		return "", "", errors.Errorf("invalid repository %q", repository)
	}
	return s[0], s[1], nil
}
