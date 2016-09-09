package catalog

import (
	"fmt"
	"log"

	"poule/gh"
	"poule/operations"
	"poule/utils"

	"github.com/google/go-github/github"
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

func (d *ciLabelAuditOperationDescriptor) OperationFromCli(*cli.Context) operations.Operation {
	return &ciLabelAuditOperation{}
}

func (d *ciLabelAuditOperationDescriptor) OperationFromConfig(operations.Configuration) operations.Operation {
	return &ciLabelAuditOperation{}
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
	pr := item.PullRequest()
	ud := userData.(ciLabelAuditOperationUserData)
	// Failing CI label but no CI failures: this is inconsistent.
	if ud.hasFailingCILabel && !ud.hasFailures {
		return fmt.Sprintf("PR#%d is labeled %q but has no failures", *pr.Number, utils.FailingCILabel)
	}
	// No failing CI label with CI failures: this is inconsistent.
	if !ud.hasFailingCILabel && ud.hasFailures {
		return fmt.Sprintf("PR#%d is not labeled %q but has failures", *pr.Number, utils.FailingCILabel)
	}
	// The pull request has a consistent combination of labels and failures.
	return ""
}

func (o *ciLabelAuditOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}) {
	// Exclude all pull requests which cannot be merged (e.g., rebase needed).
	pr := item.PullRequest()
	if pr.Mergeable != nil && !*pr.Mergeable {
		return operations.Reject, nil
	}

	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	issue, _, err := c.Client.Issues().Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	userData := ciLabelAuditOperationUserData{
		hasFailures:       utils.HasFailures(latestStatuses),
		hasFailingCILabel: utils.HasFailingCILabel(issue.Labels),
	}
	return operations.Accept, userData
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
