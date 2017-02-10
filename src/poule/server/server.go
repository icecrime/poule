package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"poule/configuration"
	"poule/operations/catalog"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	cron "gopkg.in/robfig/cron.v2"
	yaml "gopkg.in/yaml.v2"
)

const (
	// GitHubRawURLPrefix is the URL prefix for GitHub content retrieval.
	GitHubRawURLPrefix = "https://raw.githubusercontent.com"
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
func (s *Server) Run() {
	// Create and start monitoring queues.
	queues := createQueues(s.config, s)
	stopChan := monitorQueues(queues)

	// Graceful stop on SIGTERM and SIGINT.
	sigChan := make(chan os.Signal, 64)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case _, ok := <-stopChan:
			if !ok {
				return
			}
			logrus.Debug("All queues exited")
			break
		case sig := <-sigChan:
			logrus.WithField("signal", sig).Debug("received signal")
			for _, q := range queues {
				q.Consumer.Stop()
			}
			break
		}
	}
}

// FetchRepositoriesConfigs retrieves the repository specific configurations from GitHub.
func (s *Server) FetchRepositoriesConfigs() error {
	/*
		for repository := range s.config.Repositories {
			if err := s.refreshRepositoryConfiguration(repository); err != nil {
				logrus.Warnf(err.Error())
				continue
			}
		}
		return nil
	*/
	/*
		config, err := ioutil.ReadFile("/home/icecrime/go/src/github.com/docker/docker/poule.yml")
		if err != nil {
			return err
		}
		return s.updateRepositoryConfiguration("docker", config)
	*/
	return nil
}

func (s *Server) refreshRepositoryConfiguration(repository string) error {
	repoConfigFile, err := pouleConfigurationFromGitHub(repository)
	if err != nil {
		return errors.Wrapf(err, "failed to get configuration for repository %q", repository)
	}
	if len(repoConfigFile) == 0 {
		return nil
	}
	return s.updateRepositoryConfiguration(repository, repoConfigFile)
}

func (s *Server) updateRepositoryConfiguration(repository string, configFile []byte) error {
	var actions []configuration.Action
	if err := yaml.Unmarshal([]byte(configFile), &actions); err != nil {
		return errors.Wrapf(err, "failed to read configuration file for repository %q", repository)
	}

	// Stop any existing cron job for the repository.
	if c, ok := s.repositoriesConfig[repository]; ok && c.Cron != nil {
		c.Cron.Stop()
	}

	// Initialize a new cron schedule.
	repositoryCron := cron.New()
	for _, actionConfig := range actions {
		if actionConfig.Schedule != "" {
			logrus.Debugf("registering schedule %q for repository %q", actionConfig.Schedule, repository)
			repositoryCron.AddFunc(actionConfig.Schedule, func() {
				if err := executeActionOnAllItems(s.makeExecutionConfig(repository), actionConfig); err != nil {
					logrus.WithFields(logrus.Fields{
						"repository": repository,
					}).Errorf("error executing scheduled task: %v", err)
				}
			})
		}
	}
	repositoryCron.Start()

	// Store the repository specific configuration.
	s.repositoriesConfig[repository] = repositoryConfig{
		Actions: actions,
		Cron:    repositoryCron,
	}
	logrus.Infof("updated configuration for repository %q", repository)
	return nil
}

func (s *Server) makeExecutionConfig(repository string) *configuration.Config {
	return &configuration.Config{
		RunDelay:   s.config.RunDelay,
		DryRun:     s.config.DryRun,
		Token:      s.config.Token,
		TokenFile:  s.config.TokenFile,
		Repository: repository,
	}
}

func pouleConfigurationFromGitHub(repository string) ([]byte, error) {
	// Fetch a repository specific configuration from GitHub.
	configURL := fmt.Sprintf("%s/%s/master/%s", GitHubRawURLPrefix, repository, configuration.PouleConfigurationFile)
	resp, err := http.Get(configURL)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	// If the file is not found, this is not an error.
	if resp.StatusCode == http.StatusNotFound {
		logrus.Debugf("configuration file missing for repository %q", repository)
		return []byte{}, nil
	}
	return ioutil.ReadAll(resp.Body)
}
