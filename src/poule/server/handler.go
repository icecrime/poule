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

func (s *Server) HandleMessage(message *nsq.Message) error {
	// Unserialize the GitHub webhook payload into a partial message in order to inspect the type
	// of event and handle accordingly.
	var m PartialMessage
	if err := json.Unmarshal(message.Body, &m); err != nil {
		return err
	}

	// Avoid logging status event which are just too frequent and noisy.
	if m.GitHubEvent != "status" {
		logrus.WithFields(logrus.Fields{
			"action": m.Action,
			"event":  m.GitHubEvent,
		}).Debugf("received GitHub event")
	}

	// Go through the configurations that match this (event, action) couple. In the `Triggers` map,
	// keys are GitHub event types, and values are associated actions.
outer_loop:
	for _, actionConfig := range s.config.Actions {
		if actionConfig.Triggers.Contains(m.GitHubEvent, m.Action) {
			if err := s.dispatchEvent(m.GitHubEvent, message.Body, actionConfig); err != nil {
				return err
			}
			continue outer_loop
		}
	}
	return nil
}

func (s *Server) dispatchEvent(event string, data []byte, action configuration.Action) error {
	item, err := makeGitHubItem(event, data)
	switch {
	case err != nil:
		return err
	case item == nil:
		return nil
	default:
		return s.executeAction(action, *item)
	}
}

func (s *Server) executeAction(action configuration.Action, item gh.Item) error {
	// Skip the execution if the action isn't configured for that repository.
	repo := item.Repository()
	if !action.Repositories.Contains(repo) {
		logrus.Debugf("filtering event for repository=%s", repo)
		return nil
	}

	// Apply all operations on the associated repository for that item.
	for _, operationConfig := range action.Operations {
		descriptor, ok := catalog.ByNameIndex[operationConfig.Type]
		if !ok {
			return errors.Errorf("unknown operation %q", operationConfig.Type)
		}
		op, err := descriptor.OperationFromConfig(operationConfig.Settings)
		if err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"operation":  operationConfig.Type,
			"repository": repo,
		}).Info("running operation")

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
	return nil
}

func makeGitHubItem(event string, data []byte) (*gh.Item, error) {
	switch event {
	case "issues", "issue_comment":
		var evt *github.IssuesEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		// Yet another quirk of the GitHub API: the "repository" field inside
		// the issue object is nil, but not at the event root.
		evt.Issue.Repository = evt.Repo
		item := gh.MakeIssueItem(evt.Issue)
		return &item, nil
	case "pull_request", "pull_request_review", "pull_request_review_comment":
		var evt *github.PullRequestEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			return nil, err
		}
		item := gh.MakePullRequestItem(evt.PullRequest)
		return &item, nil
	default:
		return nil, nil
	}
}
