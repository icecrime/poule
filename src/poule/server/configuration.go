package server

import "poule/configuration"

type NSQConfig struct {
	LookupdAddr string   `yaml:"nsq_lookupd"`
	Channel     string   `yaml:"nsq_channel"`
	Topics      []string `yaml:"nsq_topics"`
}

// ServerConfiguration is the configuration object for the server mode.
type ServerConfiguration struct {
	configuration.Config `yaml:",inline"`
	NSQConfig            `yaml:",inline"`
	Actions              []ActionConfiguration `yaml:"configuration"`
}

// ActionConfiguration is the definition of an action: it descrbibes operations to apply on a set of
// repositories when any of the associated triggers are met.
type ActionConfiguration struct {
	// Repositories is the list of repositories full names (e.g., "docker/dokcer") that the action
	// applies to.
	Repositories StringSlice `yaml:"repositories"`

	// Triggers is the collection of GitHub events that should trigger the action. The keys must be
	// valid GitHub event types (e.g., "pull_request"), and the value must be a list of alid values
	// for the action field of the GitHub paylost (e.g., "created").
	Triggers Trigger `yaml:"triggers"`

	// Operations to apply to all repositories when any trigger is met.
	Operations []configuration.OperationConfiguration `yaml:"operations"`
}

type StringSlice []string

func (s StringSlice) Contains(item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}

type Trigger map[string]StringSlice

func (t Trigger) Contains(githubEvent, githubAction string) bool {
	if actions, ok := t[githubEvent]; ok {
		return actions.Contains(githubAction)
	}
	return false
}
