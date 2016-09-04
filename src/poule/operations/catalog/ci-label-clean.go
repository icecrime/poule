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
	registerOperation(&ciFailureLabelCleanDescriptor{})
}

type ciFailureLabelCleanDescriptor struct{}

func (d *ciFailureLabelCleanDescriptor) Description() string {
	return "clean CI failure labels"
}

func (d *ciFailureLabelCleanDescriptor) Name() string {
	return "ci-label-clean"
}

func (d *ciFailureLabelCleanDescriptor) OperationFromCli(*cli.Context) Operation {
	return &ciFailureLabelClean{}
}

func (d *ciFailureLabelCleanDescriptor) OperationFromConfig(operations.Configuration) Operation {
	return &ciFailureLabelClean{}
}

type ciFailureLabelClean struct{}

func (o *ciFailureLabelClean) Apply(c *operations.Context, pr *github.PullRequest, userData interface{}) error {
	var err error
	if hasFailures := userData.(bool); hasFailures {
		_, err = c.Client.Issues.RemoveLabelForIssue(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, utils.FailingCILabel)
	}
	return err
}

func (o *ciFailureLabelClean) Describe(c *operations.Context, pr *github.PullRequest, userData interface{}) string {
	if hasFailures := userData.(bool); hasFailures {
		return fmt.Sprintf("Removing label %q from pull request #%d", utils.FailingCILabel, *pr.Number)
	}
	return ""
}

func (o *ciFailureLabelClean) Filter(c *operations.Context, pr *github.PullRequest) (operations.FilterResult, interface{}) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	issue, _, err := c.Client.Issues.Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// Skip any issue which doesn't have a label indicating CI failure.
	if !utils.HasFailingCILabel(issue.Labels) {
		return operations.Reject, nil
	}

	// List all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Include this pull request as part of the filter, and store the failures
	// information as part of the user data.
	return operations.Accept, utils.HasFailures(latestStatuses)
}

func (o *ciFailureLabelClean) ListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
