package catalog

import (
	"errors"
	"fmt"

	"poule/configuration"
	"poule/gh"
	"poule/operations"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

// OK, global state is terrible, but I like to think of `pouleUpdaterOperation` as an exception
// rather than the norm. If more of such operations need to exist in the future, we may want to
// create a special kind of "core operations" which have privileged access to the configuration.
var PouleUpdateCallback func(repository string) error

func init() {
	registerOperation(&pouleUpdaterDescriptor{})
}

type pouleUpdaterDescriptor struct{}

func (d *pouleUpdaterDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "poule-updater",
		Description: "Update the poule configuration for the specified repository",
	}
}

func (d *pouleUpdaterDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	return nil, fmt.Errorf("The poule-updater operation cannot be created from the command line")
}

func (d *pouleUpdaterDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	return &pouleUpdaterOperation{}, nil
}

type pouleUpdaterOperation struct{}

func (o *pouleUpdaterOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *pouleUpdaterOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	if PouleUpdateCallback == nil {
		return errors.New("poule configuration update callback is nil")
	}
	return PouleUpdateCallback(item.Repository())
}

func (o *pouleUpdaterOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("updating configuration")
}

func (o *pouleUpdaterOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// We're looking for merge pull request which modify the poule configuration.
	pr := item.PullRequest
	if pr.Merged != nil && *pr.Merged == false {
		logrus.Debug("rejecting unmerged pull request")
		return operations.Reject, nil, nil
	}

	// List all files modified by the pull requests, and look for our special configuration file.
	commitFiles, _, err := c.Client.PullRequests().ListFiles(c.Username, c.Repository, item.Number(), nil)
	if err != nil {
		return operations.Reject, nil, err
	}
	for _, commitFile := range commitFiles {
		if *commitFile.Filename == configuration.PouleConfigurationFile {
			return operations.Accept, nil, nil
		}
	}
	return operations.Reject, nil, nil
}

func (o *pouleUpdaterOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return nil
}

func (o *pouleUpdaterOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return nil
}
