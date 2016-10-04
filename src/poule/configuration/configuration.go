package configuration

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// we need a special type to allow toml to decode from a duration string
// see https://github.com/BurntSushi/toml#using-the-encodingtextunmarshaler-interface
type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type Config struct {
	RunDelay   duration `toml:"delay"`
	DryRun     bool     `toml:"dry_run"`
	Repository string   `toml:"repository"`
	Token      string   `toml:"token"`
	TokenFile  string   `toml:"token_file"`
}

type OperationConfiguration struct {
	Type     string                 `toml:"type"`
	Filters  map[string]interface{} `toml:"filters"`
	Settings map[string]interface{} `toml:"settings"`
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

// SetDelay is a helper function to update the duration in the config
func (c *Config) SetDelay(t time.Duration) {
	d := duration{}
	d.Duration = t
	c.RunDelay = d
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
