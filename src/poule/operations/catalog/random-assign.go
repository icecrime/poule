package catalog

import (
	"fmt"
	"math/rand"

	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&assignDescriptor{})
}

type assignOperationConfig struct {
	Users []string `mapstructure:"users"`
}

type assignDescriptor struct{}

func (d *assignDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "random-assign",
		Description: "Assign items to a random username from the `users` list.",
		ArgsUsage:   "user [user...] ...",
	}
}

func (d *assignDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	if c.NArg() < 1 {
		return nil, errors.Errorf("random-assign requires at least one argument")
	}
	assignOperationConfig := &assignOperationConfig{Users: c.Args()}
	return d.makeAssignOperation(assignOperationConfig)
}

func (d *assignDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	assignOperationConfig := &assignOperationConfig{}
	if err := mapstructure.Decode(c, &assignOperationConfig); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	return d.makeAssignOperation(assignOperationConfig)
}

func (d *assignDescriptor) makeAssignOperation(config *assignOperationConfig) (operations.Operation, error) {
	return &assignOperation{users: config.Users}, nil
}

type assignOperation struct {
	users []string
}

func (o *assignOperation) Accepts() operations.AcceptedType {
	return operations.Issues | operations.PullRequests
}

func (o *assignOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	_, _, err := c.Client.Issues().AddAssignees(c.Username, c.Repository, item.Number(), []string{userData.(string)})
	return err
}

func (o *assignOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("assigning %s", userData.(string))
}

func (o *assignOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Reject if the item is already assigned
	if len(item.Assignees()) > 0 || item.Assignee() != nil {
		return operations.Reject, nil, nil
	}

retry:
	assignee := o.users[rand.Intn(len(o.users))]
	// Filter out author
	if item.User() != nil && assignee == *item.User().Login {
		goto retry
	}
	return operations.Accept, assignee, nil
}

func (o *assignOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func (o *assignOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
