package server

import "poule/configuration"

type ServerConfig struct {
	ListenAddr     string                `toml:"listen"`
	NSQLookupdAddr string                `toml:"nsq_lookupd_addr"`
	Topic          string                `toml:"topic"`
	Channel        string                `toml:"channel"`
	Config         *configuration.Config `toml:"config"`
	Events         []EventConfiguration  `toml:"events"`
}

type EventConfiguration struct {
	Type       string                   `toml:"type"`
	Action     string                   `toml:"action"`
	Operations []OperationConfiguration `toml:"operations"`
}

type OperationConfiguration struct {
	Type     string                 `toml:"type"`
	Filters  map[string]interface{} `toml:"filters"`
	Settings map[string]interface{} `toml:"settings"`
}
