package catalog

import (
	"fmt"

	"poule/configuration"
	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&ciLabelCleanOperationDescriptor{})
}

type ciLabelCleanOperationDescriptor struct{}

func (d *ciLabelCleanOperationDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "ci-label-clean",
		Description: "Clean CI failure labels",
	}
}

func (d *ciLabelCleanOperationDescriptor) OperationFromCli(*cli.Context) (operations.Operation, error) {
	return &ciLabelCleanOperation{}, nil
}

func (d *ciLabelCleanOperationDescriptor) OperationFromConfig(operations.Configuration) (operations.Operation, error) {
	return &ciLabelCleanOperation{}, nil
}

type ciLabelCleanOperation struct{}

func (o *ciLabelCleanOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *ciLabelCleanOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	var err error
	if hasFailures := userData.(bool); hasFailures {
		pr := item.PullRequest
		_, err = c.Client.Issues().RemoveLabelForIssue(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, configuration.FailingCILabel)
	}
	return err
}

func (o *ciLabelCleanOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	if hasFailures := userData.(bool); hasFailures {
		return fmt.Sprintf("Removing label %q", configuration.FailingCILabel)
	}
	return ""
}

func (o *ciLabelCleanOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	pr := item.PullRequest
	if _, err := item.GetRelatedIssue(c.Client); err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve issue #%d", item.Number())
	}

	// Skip any issue which doesn't have a label indicating CI failure.
	if !gh.HasFailingCILabel(item.Issue.Labels) {
		return operations.Reject, nil, nil
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve statuses for pull request #%d", *pr.Number)
	}
	latestStatuses := gh.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	return operations.Accept, latestStatuses.HasFailures(), nil
}

func (o *ciLabelCleanOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	// ciLabelCleanOperation doesn't apply to GitHub issues.
	return nil
}

func (o *ciLabelCleanOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
