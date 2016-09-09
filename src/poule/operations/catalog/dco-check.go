package catalog

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/urfave/cli"
)

var (
	dcoRegex             = regexp.MustCompile("(?m)(Docker-DCO-1.1-)?Signed-off-by: ([^<]+) <([^<>@]+@[^<>]+)>( \\(github: ([a-zA-Z0-9][a-zA-Z0-9-]+)\\))?")
	dcoCommentToken      = "AUTOMATED:POULE:DCO-EXPLANATION"
	defaultUnsignedLabel = "dco/no"
)

func init() {
	registerOperation(&dcoCheckDescriptor{})
}

type dcoCheckDescriptor struct{}

func (d *dcoCheckDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "dco-check",
		Description: "Check DCO on pull requests",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "unsigned-label",
				Usage: "label to add to unsigned pull requests",
				Value: defaultUnsignedLabel,
			},
		},
	}
}

func (d *dcoCheckDescriptor) OperationFromCli(c *cli.Context) operations.Operation {
	return &dcoCheckOperation{
		unsignedLabel: c.String("unsigned-label"),
	}
}

func (d *dcoCheckDescriptor) OperationFromConfig(c operations.Configuration) operations.Operation {
	dcoCheckOperation := &dcoCheckOperation{
		unsignedLabel: defaultUnsignedLabel,
	}
	if len(c) > 0 {
		if err := mapstructure.Decode(c, &dcoCheckOperation); err != nil {
			log.Fatalf("Error creating operation from configuration: %v", err)
		}
	}
	return dcoCheckOperation
}

type dcoCheckOperation struct {
	unsignedLabel string `mapstructure:"unsigned-label"`
}

func (o *dcoCheckOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *dcoCheckOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	fnMapping := map[bool]func(*operations.Context, *github.PullRequest) error{
		true:  o.applySigned,
		false: o.applyUnsigned,
	}
	return fnMapping[userData.(bool)](c, item.PullRequest())
}

func (o *dcoCheckOperation) applySigned(c *operations.Context, pr *github.PullRequest) error {
	// Remove the DCO unsigned label.
	if err := toggleDCOLabel(c, pr, false, o.unsignedLabel); err != nil {
		return err
	}

	// Delete the automated DCO comment (if any).
	if automatedComment, err := findDCOComment(c, pr); err != nil {
		return err
	} else if automatedComment != nil {
		if _, err := c.Client.PullRequests().DeleteComment(c.Username, c.Repository, *automatedComment.ID); err != nil {
			return err
		}
	}
	return nil
}

func (o *dcoCheckOperation) applyUnsigned(c *operations.Context, pr *github.PullRequest) error {
	// Add the DCO unsigned label.
	if err := toggleDCOLabel(c, pr, true, o.unsignedLabel); err != nil {
		return err
	}

	// Create the automated comment for that pull request, unless there is
	// already one.
	if automatedComment, err := findDCOComment(c, pr); err != nil {
		return err
	} else if automatedComment != nil {
		return nil
	}

	// Create the automated comment.
	content := formatDCOComment(c, pr)
	comment := &github.PullRequestComment{
		Body: &content,
	}
	_, _, err := c.Client.PullRequests().CreateComment(c.Username, c.Repository, *pr.Number, comment)
	return err
}

func (o *dcoCheckOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	pr := item.PullRequest()
	if isSigned := userData.(bool); isSigned {
		return fmt.Sprintf("Pull request #%d is signed: label %q and explanation comment will be removed", *pr.Number, o.unsignedLabel)
	} else {
		return fmt.Sprintf("Pull request #%d is unsigned: label %q and explanation comment will be added", *pr.Number, o.unsignedLabel)
	}
}

func (o *dcoCheckOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}) {
	// Retrieve commits for that pull request.
	pr := item.PullRequest()
	commits, _, err := c.Client.PullRequests().ListCommits(c.Username, c.Repository, *pr.Number, nil)
	if err != nil {
		log.Fatal(err)
	}

	// We take actions on every pull requests:
	//  - Those which signed get the `dco/no` label removed, as well as the
	//    comment which explains how to proceed.
	//  - Those which aren't get the `dco/no` label added, as well as the
	//    comment which explains how to proceed.
	isSigned := true
	for _, commit := range commits {
		if commit.Message != nil && !dcoRegex.MatchString(*commit.Message) {
			isSigned = false
			break
		}
	}
	return operations.Accept, isSigned
}

func (o *dcoCheckOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	// dcoCheckOperation doesn't apply to GitHub issues.
	return nil
}

func (o *dcoCheckOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func findDCOComment(c *operations.Context, pr *github.PullRequest) (*github.PullRequestComment, error) {
	// Retrieve all comments for that pull request.
	comments, _, err := c.Client.PullRequests().ListComments(c.Username, c.Repository, *pr.Number, &github.PullRequestListCommentsOptions{
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	})
	if err != nil {
		return nil, err
	}

	// Go through the comments looking for the automated token.
	for _, comment := range comments {
		if comment.Body != nil && strings.Contains(*comment.Body, dcoCommentToken) {
			return &comment, nil
		}
	}
	return nil, nil
}

func formatDCOComment(c *operations.Context, pr *github.PullRequest) string {
	comment := fmt.Sprintf("<!-- %s -->\n", dcoCommentToken)
	comment += `Please sign your commits following these rules:
https://github.com/docker/docker/blob/master/CONTRIBUTING.md#sign-your-work
The easiest way to do this is to amend the last commit:
~~~console
`
	comment += fmt.Sprintf("$ git clone -b %q %s %s\n", pr.Head.Ref, pr.Head.Repo.SSHURL, "somewhere")
	comment += "$ cd somewhere\n"
	if *pr.Commits > 1 {
		comment += fmt.Sprintf("$ git rebase -i HEAD~%d\n", pr.Commits)
		comment += "editor opens\nchange each 'pick' to 'edit'\nsave the file and quit\n"
	}
	comment += "$ git commit --amend -s --no-edit\n"
	if *pr.Commits > 1 {
		comment += "$ git rebase --continue # and repeat the amend for each commit\n"
	}
	comment += "$ git push -f\n"
	comment += `~~~

Amending updates the existing PR. You **DO NOT** need to open a new one.
`
	return comment
}

func toggleDCOLabel(c *operations.Context, pr *github.PullRequest, enable bool, label string) error {
	if enable {
		// Add unsigned label to issue.
		if _, _, err := c.Client.Issues().AddLabelsToIssue(c.Username, c.Repository, *pr.Number, []string{label}); err != nil {
			return err
		}
	} else {
		// Remove unsigned label from issue.
		if _, err := c.Client.Issues().RemoveLabelForIssue(c.Username, c.Repository, *pr.Number, label); err != nil {
			return err
		}
	}
	return nil
}
