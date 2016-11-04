package configuration

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type Config struct {
	RunDelay   time.Duration `yaml:"delay"`
	DryRun     bool          `yaml:"dry_run"`
	Repository string        `yaml:"repository"`
	Token      string        `yaml:"token"`
	TokenFile  string        `yaml:"token_file"`
}

type OperationConfiguration struct {
	Type     string                 `yaml:"type"`
	Filters  map[string]interface{} `yaml:"filters"`
	Settings map[string]interface{} `yaml:"settings"`
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

// Delay is a helper function to get the delay in a time.Duration format
func (c *Config) Delay() time.Duration {
	return time.Duration(c.RunDelay.Seconds()) * time.Second
}

func FromGlobalFlags(c *cli.Context) *Config {
	config := &Config{
		DryRun:     c.GlobalBool("dry-run"),
		Repository: c.GlobalString("repository"),
		Token:      c.GlobalString("token"),
		TokenFile:  c.GlobalString("token-file"),
	}
	return config
}

func getRepository(repository string) (string, string, error) {
	s := strings.SplitN(repository, "/", 2)
	if len(s) != 2 {
		return "", "", errors.Errorf("invalid repository %q", repository)
	}
	return s[0], s[1], nil
}
