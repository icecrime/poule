package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
)

type IssueFilter interface {
	ApplyIssue(*github.Issue) bool
}

func MakeIssueFilter(filterType, value string) (IssueFilter, error) {
	typeMapping := map[string]func(string) (IssueFilter, error){
		"assigned": makeAssignedFilter,
		"comments": asIssueFilter(makeCommentsFilter),
		"is":       asIssueFilter(makeIsFilter),
		"labels":   makeWithLabelsFilter,
		"~labels":  makeWithoutLabelsFilter,
	}
	if constructor, ok := typeMapping[filterType]; ok {
		return constructor(value)
	}
	return nil, fmt.Errorf("Unknown issue filter type %q", filterType)
}

type PullRequestFilter interface {
	ApplyPullRequest(*github.PullRequest) bool
}

func MakePullRequestFilter(filterType, value string) (PullRequestFilter, error) {
	typeMapping := map[string]func(string) (PullRequestFilter, error){
		"comments": asPullRequestFilter(makeCommentsFilter),
		"is":       asPullRequestFilter(makeIsFilter),
	}
	if constructor, ok := typeMapping[filterType]; ok {
		return constructor(value)
	}
	return nil, fmt.Errorf("Unknown pull request filter type %q", filterType)
}

// CombinedFilter can apply to both issues and pull requests.
type CombinedFilter interface {
	IssueFilter
	PullRequestFilter
}

// AssignedFilter filters issues based on whether they are assigned or not.
type AssignedFilter struct {
	isAssigned bool
}

func makeAssignedFilter(value string) (IssueFilter, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("Invalid value %q for \"assigned\" filter", value)
	}
	return AssignedFilter{b}, nil
}

func (f AssignedFilter) ApplyIssue(issue *github.Issue) bool {
	return f.isAssigned == (issue.Assignee != nil)
}

// CommentsFilter filters issues based on the number of comments.
type CommentsFilter struct {
	predicate func(int) bool
}

func makeCommentsFilter(value string) (CombinedFilter, error) {
	var count int
	var operation rune
	if n, err := fmt.Sscanf(value, "%c%d", &operation, &count); n != 2 || err != nil {
		return nil, fmt.Errorf("Invalid value %q for \"comments\" filter", value)
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
		return nil, fmt.Errorf("Bad operator %c for \"comments\" filter", operation)
	}
	return CommentsFilter{predicate}, nil
}

func (f CommentsFilter) ApplyIssue(issue *github.Issue) bool {
	return f.predicate(*issue.Comments)
}

func (f CommentsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	return f.predicate(*pullRequest.Comments)
}

// Is filters issues and pull requests based on their type.
type IsFilter struct {
	isPullRequest bool
}

func makeIsFilter(value string) (CombinedFilter, error) {
	switch value {
	case "pr":
		return IsFilter{isPullRequest: true}, nil
	case "issue":
		return IsFilter{isPullRequest: false}, nil
	default:
		return nil, fmt.Errorf("Invalid value %q for \"is\" filter", value)
	}
}

func (f IsFilter) ApplyIssue(issue *github.Issue) bool {
	// We're called on an issue: filter passes unless configured to accept pull
	// requests, and if the issue isn't really a pull request.
	return !f.isPullRequest && (issue.PullRequestLinks == nil)
}

func (f IsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	// We're called on a pull request: filter passes if configured to accept
	// pull requests.
	return f.isPullRequest
}

// WithLabelsFilter filters issues based on whether they bear all of the
// specified labels.
type WithLabelsFilter struct {
	labels []string
}

func makeWithLabelsFilter(value string) (IssueFilter, error) {
	labels := strings.Split(value, ",")
	return WithLabelsFilter{labels}, nil
}

func (f WithLabelsFilter) ApplyIssue(issue *github.Issue) bool {
	return hasAllLabels(f.labels, issue.Labels)
}

// WithoutLabelsFilter filters issues based on whether they bear none of the
// specified labels.
type WithoutLabelsFilter struct {
	labels []string
}

func makeWithoutLabelsFilter(value string) (IssueFilter, error) {
	labels := strings.Split(value, ",")
	return WithoutLabelsFilter{labels}, nil
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

// Type-casting utilities

func asIssueFilter(fn func(string) (CombinedFilter, error)) func(string) (IssueFilter, error) {
	return func(value string) (IssueFilter, error) {
		f, err := fn(value)
		return f.(IssueFilter), err
	}
}

func asPullRequestFilter(fn func(string) (CombinedFilter, error)) func(string) (PullRequestFilter, error) {
	return func(value string) (PullRequestFilter, error) {
		f, err := fn(value)
		return f.(PullRequestFilter), err
	}
}
