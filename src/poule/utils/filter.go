package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
)

type IssueFilter interface {
	Apply(*github.Issue) bool
}

func MakeIssueFilter(filterSpec string) (IssueFilter, error) {
	for prefix, constructor := range map[string]func(string) (IssueFilter, error){
		"assigned": makeAssignedApply,
		"comments": makeCommentsApply,
		"labels":   makeWithLabelsApply,
		"~labels":  makeWithoutLabelsApply,
	} {
		if strings.HasPrefix(filterSpec, prefix) {
			return constructor(filterSpec)
		}
	}
	return nil, fmt.Errorf("Unknown filter type %q", filterSpec)
}

// AssignedApply filters issues based on whether they are assigned or not.
type AssignedApply struct {
	isAssigned bool
}

func makeAssignedApply(filterSpec string) (IssueFilter, error) {
	s := strings.SplitN(filterSpec, "=", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("Bad format %q for \"assigned\" filter", filterSpec)
	}
	b, err := strconv.ParseBool(s[1])
	if err != nil {
		return nil, fmt.Errorf("Invalid value %q for \"assigned\" filter", s[1])
	}
	return AssignedApply{b}, nil
}

func (f AssignedApply) Apply(issue *github.Issue) bool {
	return f.isAssigned == (issue.Assignee != nil)
}

// CommentsApply filters issues based on the number of comments.
type CommentsApply struct {
	predicate func(int) bool
}

func makeCommentsApply(filterSpec string) (IssueFilter, error) {
	var count int
	var operation rune
	if n, err := fmt.Sscanf(filterSpec, "comments%c%d", &operation, &count); n != 2 || err != nil {
		return nil, fmt.Errorf("Bad format %q for \"comments\" filter", filterSpec)
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
	return CommentsApply{predicate}, nil
}

func (f CommentsApply) Apply(issue *github.Issue) bool {
	return f.predicate(*issue.Comments)
}

// WithLabelsApply filters issues based on whether they bear all of the
// specified labels.
type WithLabelsApply struct {
	labels []string
}

func makeWithLabelsApply(filterSpec string) (IssueFilter, error) {
	s := strings.SplitN(filterSpec, "=", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("Bad format %q for \"labels\" filter", filterSpec)
	}
	labels := strings.Split(s[1], ",")
	return WithLabelsApply{labels}, nil
}

func (f WithLabelsApply) Apply(issue *github.Issue) bool {
	return hasAllLabels(f.labels, issue.Labels)
}

// WithoutLabelsApply filters issues based on whether they bear none of the
// specified labels.
type WithoutLabelsApply struct {
	labels []string
}

func makeWithoutLabelsApply(filterSpec string) (IssueFilter, error) {
	s := strings.SplitN(filterSpec, "=", 2)
	if len(s) != 2 {
		return nil, fmt.Errorf("Bad format %q for \"~labels\" filter", filterSpec)
	}
	labels := strings.Split(s[1], ",")
	return WithoutLabelsApply{labels}, nil
}

func (f WithoutLabelsApply) Apply(issue *github.Issue) bool {
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
