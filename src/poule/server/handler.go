package server

import (
	"encoding/json"
	"poule/configuration"
	"poule/gh"
	"poule/operations"
	"poule/operations/catalog"

	"github.com/Sirupsen/logrus"
	nsq "github.com/bitly/go-nsq"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type PartialMessage struct {
	GitHubEvent    string `json:"X-GitHub-Event"`
	GitHubDelivery string `json:"X-GitHub-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
	Action         string `json:"action"`
}

func (s *Server) handler(message *nsq.Message) error {
	logrus.Debugf("nsq message: id=%s timestamp=%d", message.ID, message.Timestamp)
	for name, trigger := range s.config.Triggers {
		logrus.Debugf("processing trigger: %s", name)
		// TODO: parse repo

		if err := s.dispatchEvent(message.Body, trigger); err != nil {
			return err
		}

	}
	return nil
}

func (s *Server) dispatchEvent(data []byte, trigger TriggerConfiguration) error {
	var m PartialMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	for _, e := range trigger.Events {
		if m.GitHubEvent == e.Type && m.Action == e.Action {
			switch m.GitHubEvent {
			case "pull_request":
				if err := s.handlePullRequest(data, trigger); err != nil {
					return err
				}
			default:
				logrus.Warnf("unhandled event type: %s", m.GitHubEvent)
			}
		}
	}

	return nil
}

func (s *Server) handlePullRequest(data []byte, trigger TriggerConfiguration) error {
	var evt *github.PullRequestEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}

	logrus.Debugf("event received: repo=%s", *evt.Repo.FullName)
	for _, repo := range trigger.Repositories {
		logrus.Debugf("checking repo: %s", repo)
		if *evt.Repo.FullName == repo {
			logrus.Infof("running operations for repo: %s", repo)
			for _, operationConfig := range trigger.Operations {
				logrus.Debugf("running operation: type=%s", operationConfig.Type)
				descriptor, ok := catalog.ByNameIndex[operationConfig.Type]
				if !ok {
					return errors.Errorf("unknown operation %q", operationConfig.Type)
				}
				op, err := descriptor.OperationFromConfig(operationConfig.Settings)
				if err != nil {
					return err
				}

				item := gh.MakePullRequestItem(evt.PullRequest)
				if err := operations.RunSingle(&configuration.Config{
					RunDelay:   s.config.RunDelay,
					DryRun:     s.config.DryRun,
					Token:      s.config.Token,
					TokenFile:  s.config.TokenFile,
					Repository: repo,
				}, op, item); err != nil {
					return err
				}
			}
			break
		}
	}

	return nil
}
