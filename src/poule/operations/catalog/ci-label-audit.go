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
	registerOperation(&ciLabelAuditOperationDescriptor{})
}

type ciLabelAuditOperationDescriptor struct{}

func (d *ciLabelAuditOperationDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "ci-label-audit",
		Description: "Audit CI failure labels",
	}
}

func (d *ciLabelAuditOperationDescriptor) OperationFromCli(*cli.Context) (operations.Operation, error) {
	return &ciLabelAuditOperation{}, nil
}

func (d *ciLabelAuditOperationDescriptor) OperationFromConfig(operations.Configuration) (operations.Operation, error) {
	return &ciLabelAuditOperation{}, nil
}

type ciLabelAuditOperation struct{}

type ciLabelAuditOperationUserData struct {
	hasFailures       bool
	hasFailingCILabel bool
}

func (o *ciLabelAuditOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *ciLabelAuditOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	// Apply is a no-op for the ciLabelAuditOperation.
	return nil
}

func (o *ciLabelAuditOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	ud := userData.(ciLabelAuditOperationUserData)
	// Failing CI label but no CI failures: this is inconsistent.
	if ud.hasFailingCILabel && !ud.hasFailures {
		return fmt.Sprintf("Removing label %q", configuration.FailingCILabel)
	}
	// No failing CI label with CI failures: this is inconsistent.
	if !ud.hasFailingCILabel && ud.hasFailures {
		return fmt.Sprintf("Adding label %q", configuration.FailingCILabel)
	}
	// The pull request has a consistent combination of labels and failures.
	return ""
}

func (o *ciLabelAuditOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Exclude all pull requests which cannot be merged (e.g., rebase needed).
	pr := item.PullRequest
	if pr.Mergeable != nil && !*pr.Mergeable {
		return operations.Reject, nil, nil
	}

	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	issue, err := item.GetRelatedIssue(c.Client)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve issue #%d", *pr.Number)
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve statuses for pull request #%d", *pr.Number)
	}
	latestStatuses := gh.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	userData := ciLabelAuditOperationUserData{
		hasFailures:       latestStatuses.HasFailures(),
		hasFailingCILabel: gh.HasFailingCILabel(issue.Labels),
	}
	return operations.Accept, userData, nil
}

func (o *ciLabelAuditOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	// ciLabelAuditOperation doesn't apply to GitHub issues.
	return nil
}

func (o *ciLabelAuditOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
