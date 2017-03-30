package server

import (
	"encoding/json"
	"strings"

	"poule/configuration"
	"poule/gh"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
)

// HandleMessage handles a GitHub event.
func (s *Server) HandleMessage(event string, body []byte) error {
	// Parse into GitHub items in order to extract the repository information.
	items, err := makeGitHubItems(&s.config.Config, event, body)
	switch {
	case err != nil:
		return err
	case len(items) == 0:
		return nil
	}

	// Handle the event for every GitHub item related to this event.
	for _, item := range items {
		if err := s.handleMessageForItem(event, body, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleMessageForItem(event string, body []byte, item gh.Item) error {
	// Unserialize the body in order to extract the action.
	var m struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"action":     m.Action,
		"event":      event,
		"number":     item.Number(),
		"repository": item.Repository(),
	}).Debugf("received GitHub event")

	// Gather the list of potential actions for that repository.
	actions := s.config.CommonActions
	if repoConfig, ok := s.repositoriesConfig[item.Repository()]; ok {
		actions = append(actions, repoConfig.Actions...)
	}

	// Go through the configurations that match this (event, action) couple. In the `Triggers` map,
	// keys are GitHub event types, and values are associated actions.
outer_loop:
	for _, actionConfig := range actions {
		if actionConfig.Triggers.Contains(event, m.Action) {
			if err := executeAction(s.makeExecutionConfig(item.Repository()), actionConfig, item); err != nil {
				return err
			}
			continue outer_loop
		}
	}
	return nil
}

func makeGitHubItems(c *configuration.Config, event string, data []byte) ([]gh.Item, error) {
	switch event {
	case "issues", "issue_comment":
		return makeItemsFromIssueEvent(c, data)
	case "pull_request", "pull_request_review", "pull_request_review_comment":
		return makeItemsFromPullRequestEvent(c, data)
	// Handling of the "status" event is temporarily disabled: we don't have a use case for it yet
	// and it's extremely consuming in terms of API limits.
	//case "status":
	//	return makeItemsFromStatusEvent(c, data)
	default:
		return nil, nil
	}
}

func makeItemsFromIssueEvent(c *configuration.Config, data []byte) ([]gh.Item, error) {
	var evt *github.IssuesEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return []gh.Item{}, err
	}

	// Yet another quirk of the GitHub API: the "repository" field inside
	// the issue object is nil, but not at the event root.
	evt.Issue.Repository = evt.Repo
	item := gh.MakeIssueItem(evt.Issue)
	return []gh.Item{item}, nil
}

func makeItemsFromPullRequestEvent(c *configuration.Config, data []byte) ([]gh.Item, error) {
	var evt *github.PullRequestEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return []gh.Item{}, err
	}

	item := gh.MakePullRequestItem(evt.PullRequest)
	return []gh.Item{item}, nil
}

func makeItemsFromStatusEvent(c *configuration.Config, data []byte) ([]gh.Item, error) {
	var evt *github.StatusEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return []gh.Item{}, err
	}

	// Search for all pull request that match this commit SHA. Note that it's perfectly fine for a
	// single commit to belong to multiple pull requests (example: when a patch was cherry-picked in
	// multiple places).
	client := gh.MakeClient(c)
	result, _, err := client.Search().Issues(*evt.SHA, nil)
	if err != nil {
		return []gh.Item{}, err
	}
	logrus.Debugf("found %d matching items for SHA %s", *result.Total, *evt.SHA)

	// TODO Retrieve the commit list for the pull request, and verify that the SHA is indeed part of
	// the pull requests commits. This avoids matching pull request that contain the specified SHA
	// as part of their title or body.
	pulls := []gh.Item{}
	for _, issue := range result.Issues {
		// The issue object has an empty repository information, so we need to extract if from the
		// issue's HTML URL... <insert crying emoji here>
		if strings.HasPrefix(*issue.HTMLURL, "https://github.com/"+*evt.Repo.FullName) {
			pull, _, err := client.PullRequests().Get(*evt.Repo.Owner.Login, *evt.Repo.Name, *issue.Number)
			if err != nil {
				return []gh.Item{}, err
			}
			pulls = append(pulls, gh.MakePullRequestItem(pull))
		}
	}
	return pulls, nil
}
