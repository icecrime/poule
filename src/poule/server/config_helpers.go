package server

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"poule/configuration"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	cron "gopkg.in/robfig/cron.v2"
	yaml "gopkg.in/yaml.v2"
)

const (
	// GitHubRawURLPrefix is the URL prefix for GitHub content retrieval.
	GitHubRawURLPrefix = "https://raw.githubusercontent.com"
)

// FetchRepositoriesConfigs retrieves the repository specific configurations from GitHub.
func (s *Server) FetchRepositoriesConfigs() error {
	for repository := range s.config.Repositories {
		if err := s.refreshRepositoryConfiguration(repository); err != nil {
			logrus.Warnf(err.Error())
			continue
		}
	}
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
