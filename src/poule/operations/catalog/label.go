package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"poule/gh"
	"poule/operations"
	"poule/operations/settings"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&labelDescriptor{})
}

type labelOperationConfig struct {
	Patterns settings.MultiValuedKeys `mapstructure:"patterns"`
}

type labelDescriptor struct{}

func (d *labelDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "label",
		Description: "Apply label(s) to items which title or body matches a pattern",
		ArgsUsage:   "label=pattern[,pattern...]...",
	}
}

func (d *labelDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	if c.NArg() < 1 {
		return nil, errors.Errorf("label requires at least one argument")
	}
	patterns, err := settings.NewMultiValuedKeysFromSlice(c.Args())
	if err != nil {
		return nil, errors.Wrap(err, "parsing command line")
	}
	labelOperationConfig := &labelOperationConfig{Patterns: patterns}
	return d.makeLabelOperation(labelOperationConfig)
}

func (d *labelDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	labelOperationConfig := &labelOperationConfig{}
	if err := mapstructure.Decode(c, &labelOperationConfig); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	return d.makeLabelOperation(labelOperationConfig)
}

func (d *labelDescriptor) makeLabelOperation(config *labelOperationConfig) (operations.Operation, error) {
	patterns := map[string][]*regexp.Regexp{}
	err := config.Patterns.ForEach(func(key, value string) error {
		re, err := regexp.Compile(value)
		if err != nil {
			return errors.Wrap(err, "invalid pattern")
		}
		patterns[key] = append(patterns[key], re)
		return nil
	})
	return &labelOperation{patterns: patterns}, err
}

type labelOperation struct {
	patterns map[string][]*regexp.Regexp
}

func (o *labelOperation) Accepts() operations.AcceptedType {
	return operations.Issues | operations.PullRequests
}

func (o *labelOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	labels := userData.([]string)
	sort.Strings(labels) // Not necessary, but useful for testing
	_, _, err := c.Client.Issues().AddLabelsToIssue(c.Username, c.Repository, item.Number(), labels)
	return err
}

func (o *labelOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("adding labels %s", strings.Join(userData.([]string), ", "))
}

func (o *labelOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Try to match all provided regular expressions, and collect the set of
	// corresponding labels to apply.
	labelSet := map[string]struct{}{}
	for label, patterns := range o.patterns {
		// Skip labels we already are planning to set.
		if _, ok := labelSet[label]; ok {
			continue
		}
		// Attempt to match all regular expressions.
	PatternLoop:
		for _, pattern := range patterns {
			for _, candidate := range []string{item.Title(), item.Body()} {
				if pattern.MatchString(strings.ToLower(candidate)) {
					labelSet[label] = struct{}{}
					break PatternLoop
				}
			}
		}
	}

	// It's unnecessary to go further if there are no labels to apply.
	if len(labelSet) == 0 {
		return operations.Reject, nil, nil
	}

	// Convert the set of unique labels to a string slice.
	labels := []string{}
	for key, _ := range labelSet {
		labels = append(labels, key)
	}
	return operations.Accept, labels, nil
}

func (o *labelOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func (o *labelOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
