package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"poule/configuration"
	"poule/operations/catalog"

	"github.com/Sirupsen/logrus"
	"github.com/bitly/go-nsq"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	// GitHubRawURLPrefix is the URL prefix for GitHub content retrieval.
	GitHubRawURLPrefix = "https://raw.githubusercontent.com"
)

// Server provides operation trigger on GitHub events through a long-running job.
type Server struct {
	config             *configuration.Server
	repositoriesConfig map[string][]configuration.Action
}

// NewServer returns a new server instance.
func NewServer(config *configuration.Server) (*Server, error) {
	server := &Server{
		config:             config,
		repositoriesConfig: make(map[string][]configuration.Action),
	}

	// We initialize the special poule-updater operation which need to be given a callback into the
	// core behavior of the tool
	catalog.PouleUpdateCallback = server.updateRepositoryConfiguration
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
	for repository := range s.config.Repositories {
		if err := s.updateRepositoryConfiguration(repository); err != nil {
			logrus.Warnf(err.Error())
			continue
		}
	}
	return nil
}

// Queue represents one NSQ queue.
type Queue struct {
	Consumer *nsq.Consumer
}

// NewQueue returns a new queue instance.
func NewQueue(topic, channel, lookupd string, handler nsq.Handler) (*Queue, error) {
	logger := log.New(os.Stderr, "", log.Flags())
	consumer, err := nsq.NewConsumer(topic, channel, nsq.NewConfig())
	if err != nil {
		return nil, err
	}

	consumer.AddHandler(handler)
	consumer.SetLogger(logger, nsq.LogLevelWarning)
	if err := consumer.ConnectToNSQLookupd(lookupd); err != nil {
		return nil, err
	}

	return &Queue{Consumer: consumer}, nil
}

func createQueues(c *configuration.Server, handler nsq.Handler) []*Queue {
	// Subscribe to the message queues for each repository.
	queues := make([]*Queue, 0, len(c.Repositories))
	for _, topic := range c.Repositories {
		queue, err := NewQueue(topic, c.Channel, c.LookupdAddr, handler)
		if err != nil {
			logrus.Fatal(err)
		}
		queues = append(queues, queue)
	}
	return queues
}

func monitorQueues(queues []*Queue) <-chan struct{} {
	// Start one goroutine per queue and monitor the StopChan event.
	wg := sync.WaitGroup{}
	for _, q := range queues {
		wg.Add(1)
		go func(queue *Queue) {
			<-queue.Consumer.StopChan
			logrus.Debug("Queue stop channel signaled")
			wg.Done()
		}(q)
	}

	// Multiplex all queues exit into a single channel we can select on.
	stopChan := make(chan struct{})
	go func() {
		wg.Wait()
		stopChan <- struct{}{}
		close(stopChan)
	}()
	return stopChan
}

func (s *Server) updateRepositoryConfiguration(repository string) error {
	// Fetch a repository specific configuration from GitHub.
	repoConfigFile, err := pouleConfigurationFromGitHub(repository)
	if err != nil {
		return errors.Wrapf(err, "failed to get configuration for repository %q", repository)
	}

	// Read the YAML configuration file identified by the argument.
	if len(repoConfigFile) != 0 {
		var repoConfig []configuration.Action
		if err := yaml.Unmarshal([]byte(repoConfigFile), &repoConfig); err != nil {
			return fmt.Errorf("failed to read configuration file for repository %q: %v", repository, err)
		}

		// Store the repository specific configuration.
		s.repositoriesConfig[repository] = repoConfig
		logrus.Infof("updated configuration for repository %q from GitHub", repository)
	}
	return nil
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
