package main

import (
	"poule/configuration"
	"poule/operations"
	"poule/operations/catalog"
	"poule/operations/settings"
	"poule/runner"

	"github.com/urfave/cli"
)

func executeSingleOperation(c *cli.Context, descriptor catalog.OperationDescriptor) error {
	config := configuration.FromGlobalFlags(c)
	if err := config.Validate(); err != nil {
		return err
	}
	f, err := settings.ParseCliFilters(c)
	if err != nil {
		return err
	}
	op, err := descriptor.OperationFromCli(c)
	if err != nil {
		return err
	}
	return runSingleOperation(config, op, f)
}

func runSingleOperation(c *configuration.Config, op operations.Operation, filters settings.Filters) error {
	if filterIncludesIssues(filters) && op.Accepts()&operations.Issues == operations.Issues {
		if err := runner.RunOnEveryItem(c, op, &runner.IssueRunner{}, filters); err != nil {
			return err
		}

	}
	if filterIncludesPullRequests(filters) && op.Accepts()&operations.PullRequests == operations.PullRequests {
		if err := runner.RunOnEveryItem(c, op, &runner.PullRequestRunner{}, filters); err != nil {
			return err
		}
	}
	return nil
}

func filterIncludesIssues(filters []*settings.Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Strategy.(settings.IsFilter); ok && f.PullRequestOnly {
			return false
		}
	}
	return true
}

func filterIncludesPullRequests(filters []*settings.Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Strategy.(settings.IsFilter); ok && !f.PullRequestOnly {
			return false
		}
	}
	return true
}
