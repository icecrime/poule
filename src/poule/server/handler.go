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
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	// handle pull request
	if _, ok := payload["pull_request"]; ok {
		return s.handlePullRequest(data, trigger)
	}

	// TODO: handle issue
	logrus.Debugf("unknown event received: %+v", payload)
	return nil
}

func (s *Server) handlePullRequest(data []byte, trigger TriggerConfiguration) error {
	var evt *github.PullRequestEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}

	if *evt.PullRequest.State == "closed" {
		logrus.Debug("skipping closed PR")
		return nil
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
