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
	registerOperation(&ciLabelCleanOperationDescriptor{})
}

type ciLabelCleanOperationDescriptor struct{}

func (d *ciLabelCleanOperationDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "ci-label-clean",
		Description: "Clean CI failure labels",
	}
}

func (d *ciLabelCleanOperationDescriptor) OperationFromCli(*cli.Context) operations.Operation {
	return &ciLabelCleanOperation{}
}

func (d *ciLabelCleanOperationDescriptor) OperationFromConfig(operations.Configuration) operations.Operation {
	return &ciLabelCleanOperation{}
}

type ciLabelCleanOperation struct{}

func (o *ciLabelCleanOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *ciLabelCleanOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	var err error
	if hasFailures := userData.(bool); hasFailures {
		pr := item.PullRequest()
		_, err = c.Client.Issues().RemoveLabelForIssue(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, utils.FailingCILabel)
	}
	return err
}

func (o *ciLabelCleanOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	if hasFailures := userData.(bool); hasFailures {
		pr := item.PullRequest()
		return fmt.Sprintf("Removing label %q from pull request #%d", utils.FailingCILabel, *pr.Number)
	}
	return ""
}

func (o *ciLabelCleanOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	pr := item.PullRequest()
	issue, _, err := c.Client.Issues().Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// Skip any issue which doesn't have a label indicating CI failure.
	if !utils.HasFailingCILabel(issue.Labels) {
		return operations.Reject, nil
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	return operations.Accept, utils.HasFailures(latestStatuses)
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
