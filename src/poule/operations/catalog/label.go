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

func init() {
	registerOperation(&labelDescriptor{})
}

type labelOperationConfig struct {
	Matches labelMatchDescription `mapstructure:"matches"`
}

type labelMatchDescription map[string]string

type labelDescriptor struct{}

func (d *labelDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "label",
		Description: "Apply labels to issues and pull requests",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "match",
				Usage: "apply a label to items which body matches a pattern (format: `PATTERN:LABEL`)",
			},
		},
	}
}

func (d *labelDescriptor) OperationFromCli(c *cli.Context) operations.Operation {
	labelOperationConfig := &labelOperationConfig{
		Matches: map[string]string{},
	}
	for _, match := range c.StringSlice("match") {
		s := strings.SplitN(match, ":", 2)
		if len(s) != 2 {
			log.Fatalf("invalid match format %q", match)
		}
		labelOperationConfig.Matches[s[0]] = s[1]
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
	operation := &labelOperation{
		matches: map[*regexp.Regexp]string{},
	}
	for pattern, label := range config.Matches {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatalf("Invalid pattern %q: %v", pattern, err)
		}
		operation.matches[re] = label

	}
	return operation
}

type labelOperation struct {
	matches map[*regexp.Regexp]string
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
	labels := []string{}
	itemBody := itemBody(item)
	for re, label := range o.matches {
		if re.MatchString(itemBody) {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return operations.Reject, nil
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
