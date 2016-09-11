package catalog

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"poule/gh"
	"poule/operations"
	"poule/operations/catalog/settings"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
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
		Description: "Apply labels to issues and pull requests",
	}
}

func (d *labelDescriptor) OperationFromCli(c *cli.Context) operations.Operation {
	patterns, err := settings.NewMultiValuedKeysFromSlice(c.Args())
	if err != nil {
		log.Fatalf("Failed to create \"label\" operation; %v", err)
	}
	labelOperationConfig := &labelOperationConfig{
		Patterns: patterns,
	}
	return d.makeOperation(labelOperationConfig)
}

func (d *labelDescriptor) OperationFromConfig(c operations.Configuration) operations.Operation {
	labelOperationConfig := &labelOperationConfig{}
	if err := mapstructure.Decode(c, &labelOperationConfig); err != nil {
		log.Fatalf("Error creating operation from configuration: %v", err)
	}
	return d.makeOperation(labelOperationConfig)
}

func (d *labelDescriptor) makeOperation(config *labelOperationConfig) operations.Operation {
	patterns := map[string][]*regexp.Regexp{}
	config.Patterns.ForEach(func(key, value string) {
		re, err := regexp.Compile(value)
		if err != nil {
			log.Fatalf("Invalid value %q for pattern: %v", value, err)
		}
		patterns[key] = append(patterns[key], re)
	})
	return &labelOperation{patterns: patterns}
}

type labelOperation struct {
	patterns map[string][]*regexp.Regexp
}

func (o *labelOperation) Accepts() operations.AcceptedType {
	return operations.Issues | operations.PullRequests
}

func (o *labelOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	_, _, err := c.Client.Issues().AddLabelsToIssue(c.Username, c.Repository, itemNumber(item), []string{userData.(string)})
	return err
}

func (o *labelOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("Adding labels %s to item #%d", strings.Join(userData.([]string), ", "), itemNumber(item))
}

func (o *labelOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}) {
	itemBody := itemBody(item)

	// Try to match all provided regular expressions, and collect the set of
	// corresponding labels to apply.
	labelSet := map[string]struct{}{}
	for label, patterns := range o.patterns {
		// Skip labels we already are planning to set.
		if _, ok := labelSet[label]; ok {
			continue
		}
		// Attempt to match all regular expressions.
		for _, pattern := range patterns {
			if pattern.MatchString(itemBody) {
				labelSet[label] = struct{}{}
				break
			}
		}
	}

	// It's unnecessary to go further if there are no labels to apply.
	if len(labelSet) == 0 {
		return operations.Reject, nil
	}

	// Convert the set of unique labels to a string slice.
	labels := []string{}
	for key, _ := range labelSet {
		labels = append(labels, key)
	}
	return operations.Accept, labels
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

func itemBody(item gh.Item) string {
	switch {
	case item.IsIssue():
		return *item.Issue().Body
	case item.IsPullRequest():
		return *item.PullRequest().Body
	default:
		panic("unreachable")
	}
}

func itemNumber(item gh.Item) int {
	switch {
	case item.IsIssue():
		return *item.Issue().Number
	case item.IsPullRequest():
		return *item.PullRequest().Number
	default:
		panic("unreachable")
	}
}
