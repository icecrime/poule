package operations

import (
	"log"
	"time"

	"poule/configuration"
	"poule/utils"

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

// FilterResult describes the result of an operation filter.
type FilterResult int

const (
	// Accept means that the filter accepts the item.
	Accept FilterResult = iota

	// Reject means that the filter rejects the item
	Reject

	// Terminal means that the filter is rejected, and that no more items
	// should be sumbmitted to that filter. This is typically useful for
	// operations working on sorted sets of data, and for which the first
	// failure could also mean that no Accept may further occur.
	Terminal
)

// Configuration is an opaque data structure for operation-specific
// configuration.
type Configuration map[string]interface{}

type IssueOperation interface {
	// Apply applies the operation to the GitHub issue.
	Apply(*Context, *github.Issue, interface{}) error

	// Describe returns a human-readable description of calling Apply on the
	// specified issue with the specified context.
	Describe(*Context, *github.Issue, interface{}) string

	// Filter returns whether that operation should apply to the specified
	// issue, and an operation specific user data that is guaranteed to be
	// passed on Apply and Describe invocation.
	Filter(*Context, *github.Issue) (FilterResult, interface{})

	// ListOptions returns the global filtering options to apply when listing
	// issues for the specified context.
	ListOptions(*Context) *github.IssueListByRepoOptions
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
	Filter(*Context, *github.PullRequest) (FilterResult, interface{})

	// ListOptions returns the global filtering options to apply when listing
	// pull requests for the specified context.
	ListOptions(*Context) *github.PullRequestListOptions
}

func RunIssueOperation(c *configuration.Config, op IssueOperation) {
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
			switch filterResult, userdata := op.Filter(&context, &issue); filterResult {
			case Accept:
				if s := op.Describe(&context, &issue, userdata); s != "" {
					log.Printf(s)
				}
				if !utils.IsDryRun(c) {
					if err := op.Apply(&context, &issue, userdata); err != nil {
						log.Printf("Error applying operation on issue %d: %v", *issue.Number, err)
					}
				}
				break
			case Terminal:
				return
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if c.Delay > 0 {
			time.Sleep(c.Delay)
		}
	}
}

func RunPullRequestOperation(c *configuration.Config, op PullRequestOperation) {
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
			switch filterResult, userdata := op.Filter(&context, &pr); filterResult {
			case Accept:
				if s := op.Describe(&context, &pr, userdata); s != "" {
					log.Printf(s)
				}

				if !utils.IsDryRun(c) {
					if err := op.Apply(&context, &pr, userdata); err != nil {
						log.Printf("Error applying operation on pull request %d: %v", *pr.Number, err)
					}
				}
				break
			case Terminal:
				return
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if c.Delay > 0 {
			time.Sleep(c.Delay)
		}
	}
}
