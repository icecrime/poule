package server

import (
	"encoding/json"

	"poule/configuration"
	"poule/gh"
	"poule/runner"

	"github.com/Sirupsen/logrus"
	nsq "github.com/bitly/go-nsq"
	"github.com/google/go-github/github"
)

type partialMessage struct {
	GitHubEvent    string `json:"X-GitHub-Event"`
	GitHubDelivery string `json:"X-GitHub-Delivery"`
	HubSignature   string `json:"X-Hub-Signature"`
	Action         string `json:"action"`
}

// HandleMessage handles an NSQ message.
func (s *Server) HandleMessage(message *nsq.Message) error {
	// Unserialize the GitHub webhook payload into a partial message in order to inspect the type
	// of event and handle accordingly.
	var m partialMessage
	if err := json.Unmarshal(message.Body, &m); err != nil {
		return err
	}

	// Parse into a GitHub in order to extract the repository information.
	item, err := makeGitHubItem(m.GitHubEvent, message.Body)
	switch {
	case err != nil:
		return err
	case item == nil:
		return nil
	}

	// Avoid logging status event which are just too frequent and noisy.
	if m.GitHubEvent != "status" {
		logrus.WithFields(logrus.Fields{
			"action":     m.Action,
			"event":      m.GitHubEvent,
			"number":     item.Number(),
			"repository": item.Repository(),
		}).Debugf("received GitHub event")
	}

	// Gather the list of potential actions for that repository.
	actions := s.config.CommonActions
	if repoConfig, ok := s.repositoriesConfig[item.Repository()]; ok {
		actions = append(actions, repoConfig...)
	}

	// Go through the configurations that match this (event, action) couple. In the `Triggers` map,
	// keys are GitHub event types, and values are associated actions.
outer_loop:
	for _, actionConfig := range actions {
		if actionConfig.Triggers.Contains(m.GitHubEvent, m.Action) {
			if err := s.executeAction(actionConfig, *item); err != nil {
				return err
			}
			continue outer_loop
		}
	}
	return nil
}

func (s *Server) executeAction(action configuration.Action, item gh.Item) error {
	for _, operationConfig := range action.Operations {
		logrus.WithFields(logrus.Fields{
			"number":     item.Number(),
			"operation":  operationConfig.Type,
			"repository": item.Repository(),
		}).Info("running operation")

		config := &configuration.Config{
			RunDelay:   s.config.RunDelay,
			DryRun:     s.config.DryRun,
			Token:      s.config.Token,
			TokenFile:  s.config.TokenFile,
			Repository: item.Repository(),
		}
		if err := runner.RunSingleFromConfiguration(config, operationConfig, item); err != nil {
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
