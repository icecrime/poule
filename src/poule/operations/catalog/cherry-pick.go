package catalog

import (
	"fmt"
	"poule/gh"
	"poule/operations"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const defaultCherryPickLabel = "process/cherry-pick"

func init() {
	registerOperation(&cherryPickDescriptor{})
}

type cherryPickDescriptor struct{}

func (d *cherryPickDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "cherry-pick",
		Description: "Cherry-pick merged pull requests into a release branch",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "label",
				Usage: "label to search for and remove after cherry-pick",
				Value: defaultCherryPickLabel,
			},
		},
	}
}

func (d *cherryPickDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	return &cherryPickOperation{
		CherryPickLabel: c.String("label"),
	}, nil
}

func (d *cherryPickDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	operation := &cherryPickOperation{}
	if err := mapstructure.Decode(c, &operation); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	// Set the default value for label.
	if len(operation.CherryPickLabel) == 0 {
		operation.CherryPickLabel = defaultCherryPickLabel
	}
	return operation, nil
}

type cherryPickOperation struct {
	CherryPickLabel string `mapstructure:"label"`
}

func (o *cherryPickOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *cherryPickOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	// TODO Do the actual cherry-pick by creating (and auto-merging?) a pull request.
	// TODO On success: remove the `status/cherry-pick` label.
	// TODO On failure: leave the `status/cherry-pick` label and add a comment.
	return nil
}

func (o *cherryPickOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("cherry-picking into branch %q", userData.(string))
}

func (o *cherryPickOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// We only consider merged pull requests against the master branch which
	// have a milestone set.
	pr := item.PullRequest
	switch {
	case pr.Merged != nil && *pr.Merged == false:
		logrus.Debug("rejecting unmerged pull request")
		return operations.Reject, nil, nil
	case pr.Milestone == nil:
		logrus.Debug("rejecting pull request without milestone")
		return operations.Reject, nil, nil
	case *pr.Base.Ref != "master":
		logrus.Debugf("rejecting pull request against non-master branch %q", *pr.Base.Ref)
		return operations.Reject, nil, nil
	}

	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	if _, err := item.GetRelatedIssue(c.Client); err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve issue #%d", *pr.Number)
	} else if !gh.HasLabel(o.CherryPickLabel, item.Issue.Labels) {
		logrus.Debugf("rejecting pull request without label %q", o.CherryPickLabel)
		return operations.Reject, nil, nil
	}

	// Find a branch that corresponds to the milestone.
	branch, _, err := c.Client.Repositories().GetBranch(c.Username, c.Repository, *pr.Milestone.Title)
	if err != nil {
		return operations.Reject, nil, err
	}
	return operations.Accept, *branch.Name, nil
}

func (o *cherryPickOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return nil
}

func (o *cherryPickOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
