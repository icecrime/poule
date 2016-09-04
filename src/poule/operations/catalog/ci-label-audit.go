package catalog

import (
	"fmt"
	"log"

	"poule/operations"
	"poule/utils"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&ciFailureLabelAuditDescriptor{})
}

type ciFailureLabelAuditDescriptor struct{}

func (d *ciFailureLabelAuditDescriptor) Description() string {
	return "audit CI failure labels"
}

func (d *ciFailureLabelAuditDescriptor) Name() string {
	return "ci-label-audit"
}

func (d *ciFailureLabelAuditDescriptor) OperationFromCli(*cli.Context) Operation {
	return &ciFailureLabelAudit{}
}

func (d *ciFailureLabelAuditDescriptor) OperationFromConfig(operations.Configuration) Operation {
	return &ciFailureLabelAudit{}
}

type ciFailureLabelAudit struct{}

type ciFailureLabelAuditUserData struct {
	hasFailures       bool
	hasFailingCILabel bool
}

func (o *ciFailureLabelAudit) Apply(c *operations.Context, pr *github.PullRequest, userData interface{}) error {
	// Apply is a no-op for the ciFailureLabelAudit.
	return nil
}

func (o *ciFailureLabelAudit) Describe(c *operations.Context, pr *github.PullRequest, userData interface{}) string {
	ud := userData.(ciFailureLabelAuditUserData)
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

func (o *ciFailureLabelAudit) Filter(c *operations.Context, pr *github.PullRequest) (operations.FilterResult, interface{}) {
	// Exclude all pull requests which cannot be merged (e.g., rebase needed).
	if pr.Mergeable != nil && !*pr.Mergeable {
		return operations.Reject, nil
	}

	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	issue, _, err := c.Client.Issues.Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	userData := ciFailureLabelAuditUserData{
		hasFailures:       utils.HasFailures(latestStatuses),
		hasFailingCILabel: utils.HasFailingCILabel(issue.Labels),
	}
	return operations.Accept, userData
}

func (o *ciFailureLabelAudit) ListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
