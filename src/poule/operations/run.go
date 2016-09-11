package operations

import (
	"log"
	"time"

	"poule/configuration"
	"poule/gh"
	"poule/operations/catalog/settings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Runner interface {
	ListItems(context *Context, op Operation, page int) ([]gh.Item, *github.Response, error)
}

type IssueRunner struct{}

func (r *IssueRunner) ListItems(context *Context, op Operation, page int) ([]gh.Item, *github.Response, error) {
	// Retrieve the list options from the operation, and override the page
	// number with the current pointer.
	listOptions := op.IssueListOptions(context)
	listOptions.ListOptions.Page = page

	// List all issues for that repository with the specific settings.
	issues, resp, err := context.Client.Issues().ListByRepo(context.Username, context.Repository, listOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list issues for repository \"%s:%s\"", context.Username, context.Repository)
	}

	// Convert the result to items.
	items := []gh.Item{}
	for i, _ := range issues {
		items = append(items, gh.MakeItem(&issues[i]))
	}
	return items, resp, err
}

type PullRequestRunner struct{}

func (r *PullRequestRunner) ListItems(context *Context, op Operation, page int) ([]gh.Item, *github.Response, error) {
	// Retrieve the list options from the operation, and override the page
	// number with the current pointer.
	listOptions := op.PullRequestListOptions(context)
	listOptions.ListOptions.Page = page

	// List all issues for that repository with the specific settings.
	prs, resp, err := context.Client.PullRequests().List(context.Username, context.Repository, listOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list pull requests for repository \"%s:%s\"", context.Username, context.Repository)
	}

	// Convert the result to items.
	items := []gh.Item{}
	for i, _ := range prs {
		items = append(items, gh.MakeItem(&prs[i]))
	}
	return items, resp, err
}

func Run(c *configuration.Config, op Operation, runner Runner, filters []*settings.Filter) error {
	context := Context{}
	context.Client = gh.MakeClient(c)
	context.Username, context.Repository = c.SplitRepository()

	for page := 1; page != 0; {
		items, resp, err := runner.ListItems(&context, op, page)
		if err != nil {
			return err
		}

		// Handle each issue, filtering them using the operation first.
		for _, item := range items {
			// Apply global filters to the item.
			for _, filter := range filters {
				if filter.Apply(item) == false {
					continue
				}
			}

			// Apply operation-specific filtering.
			filterResult, userdata, err := op.Filter(&context, item)
			if err != nil {
				return err
			}

			// Proceed with operation application depending on the result of
			// the filtering.
			switch filterResult {
			case Accept:
				if s := op.Describe(&context, item, userdata); s != "" {
					log.Printf(s)
				}
				if !c.DryRun {
					if err := op.Apply(&context, item, userdata); err != nil {
						return err
					}
				}
				break
			case Terminal:
				return nil
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if c.Delay > 0 {
			time.Sleep(c.Delay)
		}
	}
	return nil
}
