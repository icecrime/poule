package server

import (
	"poule/configuration"
	"poule/operations/catalog"

	"poule/server/listeners"

	cron "gopkg.in/robfig/cron.v2"
)

type repositoryConfig struct {
	Cron    *cron.Cron
	Actions []configuration.Action
}

// Server provides operation trigger on GitHub events through a long-running job.
type Server struct {
	config             *configuration.Server
	repositoriesConfig map[string]repositoryConfig
}

// NewServer returns a new server instance.
func NewServer(config *configuration.Server) (*Server, error) {
	server := &Server{
		config:             config,
		repositoriesConfig: make(map[string]repositoryConfig),
	}

	// We initialize the special poule-updater operation which need to be given a callback into the
	// core behavior of the tool
	catalog.PouleUpdateCallback = server.refreshRepositoryConfiguration
	return server, nil
}

// Run starts the event loop, and only returns when completed.
func (s *Server) Run() error {
	// We either run in "NSQ-mode" or in direct "GitHub WebHook" mode depending on the presence of
	// the `nsq_channel` configuration key.
	var l listeners.Listener
	if s.config.HTTPListen != "" {
		l = listeners.NewGitHubListener(s.config)
	} else {
		l = listeners.NewNSQListener(s.config)
	}

	// Start the listener
	return l.Start(s)
}
