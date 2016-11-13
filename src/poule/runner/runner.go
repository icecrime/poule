package runner

import (
	"fmt"
	"time"

	"poule/configuration"
	"poule/gh"
	"poule/operations"
	"poule/operations/catalog"
	"poule/operations/settings"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

// Runner provides items for operations to run on.
type Runner interface {
	ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error)
}

// IssueRunner provides issue items for operations to run on.
type IssueRunner struct{}

// ListItems returns a list of GitHub items for the specified operation to run on.
func (r *IssueRunner) ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error) {
	// Retrieve the list options from the operation, and override the page
	// number with the current pointer.
	listOptions := op.IssueListOptions(context)
	if listOptions == nil {
		return nil, nil, fmt.Errorf("operation doesn't provide list options for issues")
	}
	listOptions.ListOptions.Page = page

	// List all issues for that repository with the specific settings.
	issues, resp, err := context.Client.Issues().ListByRepo(context.Username, context.Repository, listOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list issues for repository \"%s:%s\"", context.Username, context.Repository)
	}

	// Convert the result to items.
	items := []gh.Item{}
	for i := range issues {
		items = append(items, gh.MakeIssueItem(issues[i]))
	}
	return items, resp, err
}

// PullRequestRunner provides pull request items for operations to run on.
type PullRequestRunner struct{}

// ListItems returns a list of GitHub items for the specified operation to run on.
func (r *PullRequestRunner) ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error) {
	// Retrieve the list options from the operation, and override the page
	// number with the current pointer.
	listOptions := op.PullRequestListOptions(context)
	if listOptions == nil {
		return nil, nil, fmt.Errorf("operation doesn't provide list options for pull requests")
	}
	listOptions.ListOptions.Page = page

	// List all issues for that repository with the specific settings.
	prs, resp, err := context.Client.PullRequests().List(context.Username, context.Repository, listOptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list pull requests for repository \"%s:%s\"", context.Username, context.Repository)
	}

	// Convert the result to items.
	items := []gh.Item{}
	for i := range prs {
		items = append(items, gh.MakePullRequestItem(prs[i]))
	}
	return items, resp, err
}

// RunOnEveryItem runs the specified operation on all known items as provided by the specified
// runner.
func RunOnEveryItem(c *configuration.Config, op operations.Operation, runner Runner, filters settings.Filters) error {
	context := operations.Context{}
	context.Client = gh.MakeClient(c)
	context.Username, context.Repository = c.SplitRepository()

	for page := 1; page != 0; {
		items, resp, err := runner.ListItems(&context, op, page)
		if err != nil {
			return err
		}

		// Handle each issue, filtering them using the operation first.
		for _, item := range items {
			if err := RunSingle(c, op, item, filters); err != nil {
				return err
			}
		}

		// Move on to the next page, and respect the specified delay to avoid
		// hammering the GitHub API.
		page = resp.NextPage
		if c.Delay() > 0 {
			time.Sleep(c.Delay())
		}
	}
	return nil
}

// RunSingle runs the specified operations on a single GitHub item.
func RunSingle(c *configuration.Config, op operations.Operation, item gh.Item, filters settings.Filters) error {
	context := operations.Context{}
	context.Client = gh.MakeClient(c)
	context.Username, context.Repository = c.SplitRepository()

	// Apply global filters to the item.
	if !filters.Apply(item) {
		return nil
	}

	// Apply operation-specific filtering.
	filterResult, userdata, err := op.Filter(&context, item)
	if err != nil {
		return err
	}

	// Proceed with operation application depending on the result of
	// the filtering.
	switch filterResult {
	case operations.Accept:
		if s := op.Describe(&context, item, userdata); s != "" {
			logrus.WithFields(logrus.Fields{
				"dry_run":    c.DryRun,
				"item_num":   item.Number(),
				"item_type":  item.Type(),
				"repository": c.Repository,
			}).Info(s)
		}
		if !c.DryRun {
			if err := op.Apply(&context, item, userdata); err != nil {
				return err
			}
		}
		break
	case operations.Terminal:
		return nil
	}

	return nil
}

// RunSingleFromConfiguration runs the operations as described by its configuration on a single
// GitHub item.
func RunSingleFromConfiguration(c *configuration.Config, operationConfig configuration.OperationConfiguration, item gh.Item) error {
	// Run the filters first: there's no need to go further if the filters are rejecting the item
	// anyway.
	if itemFilters, err := settings.ParseConfigurationFilters(operationConfig.Filters); err != nil {
		return err
	} else if !itemFilters.Apply(item) {
		return nil
	}

	// Create and execute the operation. Note that we pass an empty set of filters, as we have
	// already run them before.
	descriptor, ok := catalog.ByNameIndex[operationConfig.Type]
	if !ok {
		return errors.Errorf("unknown operation %q", operationConfig.Type)
	}
	op, err := descriptor.OperationFromConfig(operationConfig.Settings)
	if err != nil {
		return err
	}
	return RunSingle(c, op, item, settings.Filters{})
}
