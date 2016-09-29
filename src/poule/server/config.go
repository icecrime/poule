package server

import "time"

type ServerConfig struct {
	ListenAddr     string        `toml:"listen"`
	NSQLookupdAddr string        `toml:"nsq_lookupd_addr"`
	Topic          string        `toml:"topic"`
	Channel        string        `toml:"channel"`
	DryRun         bool          `toml:"dry_run"`
	Delay          time.Duration `toml:"delay"`
	Token          string        `toml:"token"`
	TokenFile      string        `toml:"token_file"`

	Triggers map[string]TriggerConfiguration `toml:"triggers"`
}

type TriggerConfiguration struct {
	Repositories []string                 `toml:"repositories"`
	Operations   []OperationConfiguration `toml:"operations"`
}

type OperationConfiguration struct {
	Type     string                 `toml:"type"`
	Filters  map[string]interface{} `toml:"filters"`
	Settings map[string]interface{} `toml:"settings"`
}
