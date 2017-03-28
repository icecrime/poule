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

// OperationRunner is responsible for executing the Operation.
type OperationRunner struct {
	// Config is the global settings for execution.
	Config *configuration.Config

	// GlobalFilters are the filters to apply in addition to the operation's specific filtering.
	GlobalFilters settings.Filters

	// Operation is the operation to execute.
	Operation operations.Operation
}

// NewOperationRunner returns an OperationRunner.
func NewOperationRunner(config *configuration.Config, operation operations.Operation) *OperationRunner {
	return &OperationRunner{
		Config:    config,
		Operation: operation,
	}
}

// NewOperationRunnerFromConfig returns an OperationRunner parsed from configuration.
func NewOperationRunnerFromConfig(config *configuration.Config, operationConfig *configuration.OperationConfiguration) (*OperationRunner, error) {
	// Parse the filters.
	filters, err := settings.ParseConfigurationFilters(operationConfig.Filters)
	if err != nil {
		return nil, err
	}

	// Create the operation.
	operation, err := catalog.OperationFromConfig(operationConfig)
	if err != nil {
		return nil, err
	}

	// Return a fully ready OperationRunner.
	return &OperationRunner{
		Config:        config,
		GlobalFilters: filters,
		Operation:     operation,
	}, nil
}

// Handle applies the operation to a single GitHub item.
func (r *OperationRunner) Handle(item gh.Item) error {
	return runSingle(r.Config, r.Operation, item, r.GlobalFilters)
}

// HandleStock applies the operation to the entire stock of GitHub items.
func (r *OperationRunner) HandleStock() error {
	if settings.FilterIncludesIssues(r.GlobalFilters) && r.Operation.Accepts()&operations.Issues == operations.Issues {
		if err := runOnEveryItem(r.Config, r.Operation, &IssueLister{}, r.GlobalFilters); err != nil {
			return err
		}

	}
	if settings.FilterIncludesPullRequests(r.GlobalFilters) && r.Operation.Accepts()&operations.PullRequests == operations.PullRequests {
		if err := runOnEveryItem(r.Config, r.Operation, &PullRequestLister{}, r.GlobalFilters); err != nil {
			return err
		}
	}
	return nil
}

// Lister provides items for operations to run on.
type Lister interface {
	ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error)
}

// IssueLister provides issue items for operations to run on.
type IssueLister struct{}

// ListItems returns a list of GitHub items for the specified operation to run on.
func (r *IssueLister) ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error) {
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

// PullRequestLister provides pull request items for operations to run on.
type PullRequestLister struct{}

// ListItems returns a list of GitHub items for the specified operation to run on.
func (r *PullRequestLister) ListItems(context *operations.Context, op operations.Operation, page int) ([]gh.Item, *github.Response, error) {
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

// runSingle runs the specified operations on a single GitHub item.
func runSingle(c *configuration.Config, op operations.Operation, item gh.Item, filters settings.Filters) error {
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

// runOnEveryItem runs the specified operation on all known items as provided by the specified
// lister.
func runOnEveryItem(c *configuration.Config, op operations.Operation, lister Lister, filters settings.Filters) error {
	context := operations.Context{}
	context.Client = gh.MakeClient(c)
	context.Username, context.Repository = c.SplitRepository()

	for page := 1; page != 0; {
		items, resp, err := lister.ListItems(&context, op, page)
		if err != nil {
			return err
		}

		// Handle each issue, filtering them using the operation first.
		for _, item := range items {
			if err := runSingle(c, op, item, filters); err != nil {
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
