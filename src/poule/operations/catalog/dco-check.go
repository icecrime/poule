package catalog

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	dcoContext      = "dco-signed"
	dcoCommentToken = "AUTOMATED:POULE:DCO-EXPLANATION"
	dcoRegex        = regexp.MustCompile("(?m)(Docker-DCO-1.1-)?Signed-off-by: ([^<]+) <([^<>@]+@[^<>]+)>( \\(github: ([a-zA-Z0-9][a-zA-Z0-9-]+)\\))?")
	dcoURL          = "https://github.com/docker/docker/blob/master/CONTRIBUTING.md#sign-your-work"

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

func (d *dcoCheckDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	return &dcoCheckOperation{
		UnsignedLabel: c.String("unsigned-label"),
	}, nil
}

func (d *dcoCheckDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	dcoCheckOperation := &dcoCheckOperation{}
	if len(c) > 0 {
		if err := mapstructure.Decode(c, &dcoCheckOperation); err != nil {
			return nil, errors.Wrap(err, "decoding configuration")
		}
	}
	if dcoCheckOperation.UnsignedLabel == "" {
		dcoCheckOperation.UnsignedLabel = defaultUnsignedLabel
	}
	return dcoCheckOperation, nil
}

type dcoCheckOperation struct {
	UnsignedLabel string `mapstructure:"unsigned-label"`
}

func (o *dcoCheckOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *dcoCheckOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	fnMapping := map[bool]func(*operations.Context, *github.PullRequest) error{
		true:  o.applySigned,
		false: o.applyUnsigned,
	}
	return fnMapping[userData.(bool)](c, item.PullRequest)
}

func (o *dcoCheckOperation) applySigned(c *operations.Context, pr *github.PullRequest) error {
	// Remove the DCO unsigned label.
	if err := toggleDCOLabel(c, pr, false, o.UnsignedLabel); err != nil {
		return err
	}

	// Delete the automated DCO comment (if any).
	automatedComments, err := findDCOComments(c, pr)
	if err != nil {
		return err
	}
	for _, comment := range automatedComments {
		if _, err := c.Client.Issues().DeleteComment(c.Username, c.Repository, *comment.ID); err != nil {
			return err
		}
	}

	// Set the status as successful.
	_, _, err = c.Client.Repositories().CreateStatus(c.Username, c.Repository, *pr.Head.SHA, &github.RepoStatus{
		Context:     github.String(dcoContext),
		Description: github.String("All commits are signed"),
		State:       github.String("success"),
	})
	return err
}

func (o *dcoCheckOperation) applyUnsigned(c *operations.Context, pr *github.PullRequest) error {
	// Add the DCO unsigned label.
	if err := toggleDCOLabel(c, pr, true, o.UnsignedLabel); err != nil {
		return err
	}

	// Create the automated comment for that pull request, unless there is
	// already one.
	if automatedComments, err := findDCOComments(c, pr); err != nil {
		return err
	} else if len(automatedComments) != 0 {
		return nil
	}

	// Create the automated comment.
	content := formatDCOComment(c, pr)
	comment := &github.IssueComment{Body: &content}
	if _, _, err := c.Client.Issues().CreateComment(c.Username, c.Repository, *pr.Number, comment); err != nil {
		return err
	}

	// Set the status as failing.
	_, _, err := c.Client.Repositories().CreateStatus(c.Username, c.Repository, *pr.Head.SHA, &github.RepoStatus{
		Context:     github.String(dcoContext),
		Description: github.String("Some commits don't have signature"),
		State:       github.String("failure"),
		TargetURL:   github.String(dcoURL),
	})
	return err
}

func (o *dcoCheckOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	if isSigned := userData.(bool); isSigned {
		return fmt.Sprintf("pull request is signed: removing label %q and explanation comment", o.UnsignedLabel)
	}
	return fmt.Sprintf("pull request is unsigned: adding label %q and explanation comment", o.UnsignedLabel)
}

func (o *dcoCheckOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Retrieve commits for that pull request.
	pr := item.PullRequest
	commits, _, err := c.Client.PullRequests().ListCommits(c.Username, c.Repository, *pr.Number, nil)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve commits for pull request #%d", *pr.Number)
	}

	// We take actions on every pull requests:
	//  - Those which signed get the `dco/no` label removed, as well as the
	//    comment which explains how to proceed.
	//  - Those which aren't get the `dco/no` label added, as well as the
	//    comment which explains how to proceed.
	isSigned := true
	for _, commit := range commits {
		if commit.Commit != nil && !dcoRegex.MatchString(*commit.Commit.Message) {
			isSigned = false
			break
		}
	}
	return operations.Accept, isSigned, nil
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

func findDCOComments(c *operations.Context, pr *github.PullRequest) ([]*github.IssueComment, error) {
	automatedComments := []*github.IssueComment{}
	issuesListOptions := &github.IssueListCommentsOptions{
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}

	// Retrieve all comments for that pull request.
	for page := 1; page != 0; {
		issuesListOptions.ListOptions.Page = page
		comments, resp, err := c.Client.Issues().ListComments(c.Username, c.Repository, *pr.Number, issuesListOptions)
		if err != nil {
			return nil, err
		}

		// Go through the comments looking for the automated token.
		for i := range comments {
			comment := comments[i]
			if comment.Body != nil && strings.Contains(*comment.Body, dcoCommentToken) {
				automatedComments = append(automatedComments, comment)
			}
		}

		page = resp.NextPage
	}
	return automatedComments, nil
}

func formatDCOComment(c *operations.Context, pr *github.PullRequest) string {
	comment := fmt.Sprintf("<!-- %s -->\n", dcoCommentToken)
	comment += `Please sign your commits following these rules:
https://github.com/docker/docker/blob/master/CONTRIBUTING.md#sign-your-work
The easiest way to do this is to amend the last commit:
~~~console
`
	comment += fmt.Sprintf("$ git clone -b %q %s %s\n", *pr.Head.Ref, *pr.Head.Repo.SSHURL, "somewhere")
	comment += "$ cd somewhere\n"
	if pr.Commits != nil && *pr.Commits > 1 {
		comment += fmt.Sprintf("$ git rebase -i HEAD~%d\n", pr.Commits)
		comment += "editor opens\nchange each 'pick' to 'edit'\nsave the file and quit\n"
	}
	comment += "$ git commit --amend -s --no-edit\n"
	if pr.Commits != nil && *pr.Commits > 1 {
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
		if resp, err := c.Client.Issues().RemoveLabelForIssue(c.Username, c.Repository, *pr.Number, label); err != nil {
			// Ignore 404 errors.
			if resp.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}
	}
	return nil
}
