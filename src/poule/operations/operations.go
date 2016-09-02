package operations

import (
	"log"
	"time"

	"poule/utils"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

type Context struct {
	// Client is the GitHub API client instance.
	Client *github.Client

	// Username is the owner of the GitHub repository.
	Username string

	// Repository is the name of the GitHub repository.
	Repository string
}

type IssueOperation interface {
	// Apply applies the operation to the GitHub issue.
	Apply(*Context, *github.Issue, interface{}) error

	// Describe returns a human-readable description of calling Apply on the
	// specified issue with the specified context.
	Describe(*Context, *github.Issue, interface{}) string

	// Filter returns whether that operation should apply to the specified
	// issue, and an operation specific user data that is guaranteed to be
	// passed on Apply and Describe invocation.
	Filter(*Context, *github.Issue) (bool, interface{})

	// ListOptions returns the global filtering options to apply when listing
	// issues for the specified context.
	ListOptions(*Context) *github.IssueListByRepoOptions
}

func RunIssueOperation(c *cli.Context, op IssueOperation) {
	context := Context{}
	context.Client = utils.MakeGitHubClient(c)
	context.Username, context.Repository = utils.GetRepository(c)

	for page := 1; page != 0; {
		// Retrieve the list options from the operation, and override the page
		// number with the current pointer.
		listOptions := op.ListOptions(&context)
		listOptions.ListOptions.Page = page

		// List all issues for that repository with the specific settings.
		issues, resp, err := context.Client.Issues.ListByRepo(context.Username, context.Repository, listOptions)
		if err != nil {
			log.Fatal(err)
		}

		// Handle each issue, filtering them using the operation first.
		for _, issue := range issues {
			if ok, userdata := op.Filter(&context, &issue); ok {
				if s := op.Describe(&context, &issue, userdata); s != "" {
					log.Printf(s)
				}

				if !utils.IsDryRun(c) {
					if err := op.Apply(&context, &issue, userdata); err != nil {
						log.Printf("Error applying operation on issue %d: %v", *issue.Number, err)
					}
				}
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if delay := c.GlobalDuration("delay"); delay > 0 {
			time.Sleep(delay)
		}
	}
}

type PullRequestOperation interface {
	// Apply applies the operation to the GitHub pull request.
	Apply(*Context, *github.PullRequest, interface{}) error

	// Describe returns a human-readable description of calling Apply on the
	// specified pull request with the specified context.
	Describe(*Context, *github.PullRequest, interface{}) string

	// Filter returns whether that operation should apply to the specified
	// pull request, and an operation specific user data that is guaranteed to
	// be passed on Apply and Describe invocation.
	Filter(*Context, *github.PullRequest) (bool, interface{})

	// ListOptions returns the global filtering options to apply when listing
	// pull requests for the specified context.
	ListOptions(*Context) *github.PullRequestListOptions
}

func RunPullRequestOperation(c *cli.Context, op PullRequestOperation) {
	context := Context{}
	context.Client = utils.MakeGitHubClient(c)
	context.Username, context.Repository = utils.GetRepository(c)

	for page := 1; page != 0; {
		// Retrieve the list options from the operation, and override the page
		// number with the current pointer.
		listOptions := op.ListOptions(&context)
		listOptions.ListOptions.Page = page

		// List all issues for that repository with the specific settings.
		prs, resp, err := context.Client.PullRequests.List(context.Username, context.Repository, listOptions)
		if err != nil {
			log.Fatal(err)
		}

		// Handle each issue, filtering them using the operation first.
		for _, pr := range prs {
			if ok, userdata := op.Filter(&context, &pr); ok {
				if s := op.Describe(&context, &pr, userdata); s != "" {
					log.Printf(s)
				}

				if !utils.IsDryRun(c) {
					if err := op.Apply(&context, &pr, userdata); err != nil {
						log.Printf("Error applying operation on pull request %d: %v", *pr.Number, err)
					}
				}
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if delay := c.GlobalDuration("delay"); delay > 0 {
			time.Sleep(delay)
		}
	}
}
