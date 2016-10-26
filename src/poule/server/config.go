package server

import "poule/configuration"

type ServerConfig struct {
	configuration.Config
	ListenAddr     string                          `toml:"listen_addr"`
	NSQLookupdAddr string                          `toml:"nsq_lookupd_addr"`
	Topic          string                          `toml:"topic"`
	Channel        string                          `toml:"channel"`
	Triggers       map[string]TriggerConfiguration `toml:"triggers"`
}

type Event struct {
	Type   string `toml:"type"`
	Action string `toml:"action"`
}

type TriggerConfiguration struct {
	Repositories []string                               `toml:"repositories"`
	Events       []Event                                `toml:"events"`
	Operations   []configuration.OperationConfiguration `toml:"operations"`
}
