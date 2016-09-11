package utils

import (
	"fmt"
	"strconv"
	"strings"

	"poule/gh"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Filter struct {
	Impl interface{}
}

func (f *Filter) Apply(item gh.Item) bool {
	switch {
	case item.IsIssue():
		if f, ok := f.Impl.(issueFilter); ok {
			return f.ApplyIssue(item.Issue())
		}
	case item.IsPullRequest():
		if f, ok := f.Impl.(pullRequestFilter); ok {
			return f.ApplyPullRequest(item.PullRequest())
		}
	default:
		panic("unreachable")
	}
	return true
}

type issueFilter interface {
	ApplyIssue(*github.Issue) bool
}

type pullRequestFilter interface {
	ApplyPullRequest(*github.PullRequest) bool
}

func MakeFilter(filterType, value string) (*Filter, error) {
	typeMapping := map[string]func(string) (*Filter, error){
		"assigned": makeAssignedFilter,
		"comments": makeCommentsFilter,
		"is":       makeIsFilter,
		"labels":   makeWithLabelsFilter,
		"~labels":  makeWithoutLabelsFilter,
	}
	if constructor, ok := typeMapping[filterType]; ok {
		return constructor(value)
	}
	return nil, errors.Errorf("unknown filter type %q", filterType)
}

// AssignedFilter filters issues based on whether they are assigned or not.
type AssignedFilter struct {
	isAssigned bool
}

func makeAssignedFilter(value string) (*Filter, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return nil, errors.Errorf("invalid value %q for \"assigned\" filter", value)
	}
	return asFilter(AssignedFilter{b}), nil
}

func (f AssignedFilter) ApplyIssue(issue *github.Issue) bool {
	return f.isAssigned == (issue.Assignee != nil)
}

// CommentsFilter filters issues based on the number of comments.
type CommentsFilter struct {
	predicate func(int) bool
}

func makeCommentsFilter(value string) (*Filter, error) {
	var count int
	var operation rune
	if n, err := fmt.Sscanf(value, "%c%d", &operation, &count); n != 2 || err != nil {
		return nil, errors.Errorf("invalid value %q for \"comments\" filter", value)
	}

	var predicate func(int) bool
	switch operation {
	case '<':
		predicate = func(n int) bool { return n < count }
		break
	case '=':
		predicate = func(n int) bool { return n == count }
		break
	case '>':
		predicate = func(n int) bool { return n > count }
		break
	default:
		return nil, errors.Errorf("invalid operator %c for \"comments\" filter", operation)
	}
	return asFilter(CommentsFilter{predicate}), nil
}

func (f CommentsFilter) ApplyIssue(issue *github.Issue) bool {
	return f.predicate(*issue.Comments)
}

func (f CommentsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	return f.predicate(*pullRequest.Comments)
}

// Is filters issues and pull requests based on their type.
type IsFilter struct {
	PullRequestOnly bool
}

func makeIsFilter(value string) (*Filter, error) {
	switch value {
	case "pr":
		return asFilter(IsFilter{PullRequestOnly: true}), nil
	case "issue":
		return asFilter(IsFilter{PullRequestOnly: false}), nil
	default:
		return nil, errors.Errorf("invalid value %q for \"is\" filter", value)
	}
}

func (f IsFilter) ApplyIssue(issue *github.Issue) bool {
	// We're called on an issue: filter passes unless configured to accept pull
	// requests, and if the issue isn't really a pull request.
	return !f.PullRequestOnly && (issue.PullRequestLinks == nil)
}

func (f IsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	// We're called on a pull request: filter passes if configured to accept
	// pull requests.
	return f.PullRequestOnly
}

// WithLabelsFilter filters issues based on whether they bear all of the
// specified labels.
type WithLabelsFilter struct {
	labels []string
}

func makeWithLabelsFilter(value string) (*Filter, error) {
	labels := strings.Split(value, ",")
	return asFilter(WithLabelsFilter{labels}), nil
}

func (f WithLabelsFilter) ApplyIssue(issue *github.Issue) bool {
	return hasAllLabels(f.labels, issue.Labels)
}

// WithoutLabelsFilter filters issues based on whether they bear none of the
// specified labels.
type WithoutLabelsFilter struct {
	labels []string
}

func makeWithoutLabelsFilter(value string) (*Filter, error) {
	labels := strings.Split(value, ",")
	return asFilter(WithoutLabelsFilter{labels}), nil
}

func (f WithoutLabelsFilter) ApplyIssue(issue *github.Issue) bool {
	return !hasAnyLabels(f.labels, issue.Labels)
}

func hasLabel(label string, issueLabels []github.Label) bool {
	for _, issueLabel := range issueLabels {
		if label == *issueLabel.Name {
			return true
		}
	}
	return false
}

func hasAnyLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if hasLabel(label, issueLabels) {
			return true
		}
	}
	return false
}

func hasAllLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if !hasLabel(label, issueLabels) {
			return false
		}
	}
	return true
}

// Type conversion utility.
func asFilter(impl interface{}) *Filter {
	return &Filter{
		Impl: impl,
	}
}
