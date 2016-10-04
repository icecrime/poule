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

type TriggerConfiguration struct {
	Repositories []string                               `toml:"repositories"`
	Operations   []configuration.OperationConfiguration `toml:"operations"`
}
