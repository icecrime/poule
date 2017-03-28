package settings

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"poule/gh"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const filterFlagName = "filter"

// FilteringFlag is the command-line flag that gets automatically added to
// every available operation.
var FilteringFlag = cli.StringSliceFlag{
	Name:  filterFlagName,
	Usage: "filter based on item attributes",
}

// ParseCliFilters reads filter definitions from the command line.
func ParseCliFilters(c *cli.Context) ([]*Filter, error) {
	value, err := NewMultiValuedKeysFromSlice(c.StringSlice(filterFlagName))
	if err != nil {
		return nil, err
	}
	return ParseConfigurationFilters(value.ToSerializedFormat())
}

// ParseConfigurationFilters reads filter definitions from the serialized
// configuration format.
func ParseConfigurationFilters(values map[string]interface{}) (Filters, error) {
	filters := Filters{}
	for filterType, rawValue := range values {
		value, err := filterValue(rawValue)
		if err != nil {
			return []*Filter{}, err
		}
		filter, err := MakeFilter(filterType, value)
		if err != nil {
			return []*Filter{}, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

// Filters is a collection of Filter instances.
type Filters []*Filter

// Apply returns true only if all filters accept the item.
func (f Filters) Apply(item gh.Item) bool {
	for _, filter := range f {
		if !filter.Apply(item) {
			return false
		}
	}
	return true
}

// Filter accepts or rejects a GitHub item based on a strategy.
type Filter struct {
	Strategy fmt.Stringer
}

// Apply returns whether the internal strategy is accepting or rejecting the
// specified GitHub item.
func (f *Filter) Apply(item gh.Item) bool {
	switch {
	case item.IsIssue():
		if f, ok := f.Strategy.(issueFilter); ok {
			return f.ApplyIssue(item.Issue)
		}
	case item.IsPullRequest():
		if f, ok := f.Strategy.(pullRequestFilter); ok {
			return f.ApplyPullRequest(item.PullRequest)
		}
	default:
		panic("unreachable")
	}
	return true
}

// issueFilter is a filtering strategy that applies to GitHub issues.
type issueFilter interface {
	ApplyIssue(*github.Issue) bool
	String() string
}

// pullRequestFilter is a filtering strategy that applies to GitHub pull
// requests.
type pullRequestFilter interface {
	ApplyPullRequest(*github.PullRequest) bool
	String() string
}

// MakeFilter creates a filter from a type identifier and a string value.
func MakeFilter(filterType, value string) (*Filter, error) {
	typeMapping := map[string]func(string) (*Filter, error){
		"age":      makeAgeFilter,
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

// AgeFilter filters items based on their age.
type AgeFilter struct {
	age ExtDuration
}

func makeAgeFilter(value string) (*Filter, error) {
	d, err := ParseExtDuration(value)
	if err != nil {
		return nil, errors.Errorf("invalid value %q for \"age\" filter", value)
	}
	return asFilter(AgeFilter{d}), nil
}

// ApplyIssue applies the filter to the specified issue.
func (f AgeFilter) ApplyIssue(issue *github.Issue) bool {
	return time.Since(*issue.CreatedAt) > f.age.Duration()
}

// ApplyPullRequest applies the filter to the specified pull request.
func (f AgeFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	return time.Since(*pullRequest.CreatedAt) > f.age.Duration()
}

// String returns a string representation of the filter
func (f AgeFilter) String() string {
	return fmt.Sprintf("AgeFilter(%s)", f.age)
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

// ApplyIssue applies the filter to the specified issue.
func (f AssignedFilter) ApplyIssue(issue *github.Issue) bool {
	return f.isAssigned == (issue.Assignee != nil)
}

// String returns a string representation of the filter
func (f AssignedFilter) String() string {
	return fmt.Sprintf("AssignedFilter(%t)", f.isAssigned)
}

// CommentsFilter filters issues based on the number of comments.
type CommentsFilter struct {
	filtValue string
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
	return asFilter(CommentsFilter{
		filtValue: value,
		predicate: predicate,
	}), nil
}

// ApplyIssue applies the filter to the specified issue.
func (f CommentsFilter) ApplyIssue(issue *github.Issue) bool {
	return f.predicate(*issue.Comments)
}

// String returns a string representation of the filter
func (f CommentsFilter) String() string {
	return fmt.Sprintf("CommentsFilter(%s)", f.filtValue)
}

// ApplyPullRequest applies the filter to the specified pull request.
func (f CommentsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	return f.predicate(*pullRequest.Comments)
}

// IsFilter filters issues and pull requests based on their type.
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

// ApplyIssue applies the filter to the specified issue.
func (f IsFilter) ApplyIssue(issue *github.Issue) bool {
	// We're called on an issue: filter passes unless configured to accept pull
	// requests, and if the issue isn't really a pull request.
	return !f.PullRequestOnly && (issue.PullRequestLinks == nil)
}

// ApplyPullRequest applies the filter to the specified pull request.
func (f IsFilter) ApplyPullRequest(pullRequest *github.PullRequest) bool {
	// We're called on a pull request: filter passes if configured to accept
	// pull requests.
	return f.PullRequestOnly
}

// String returns a string representation of the filter
func (f IsFilter) String() string {
	return fmt.Sprintf("IsFilter(PullRequestOnly=%t)", f.PullRequestOnly)
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

// ApplyIssue applies the filter to the specified issue.
func (f WithLabelsFilter) ApplyIssue(issue *github.Issue) bool {
	return gh.HasAllLabels(f.labels, issue.Labels)
}

// String returns a string representation of the filter
func (f WithLabelsFilter) String() string {
	return fmt.Sprintf("WithLabelsFilter(%s)", strings.Join(f.labels, ","))
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

// ApplyIssue applies the filter to the specified issue.
func (f WithoutLabelsFilter) ApplyIssue(issue *github.Issue) bool {
	return !gh.HasAnyLabels(f.labels, issue.Labels)
}

// String returns a string representation of the filter
func (f WithoutLabelsFilter) String() string {
	return fmt.Sprintf("WithoutLabelsFilter(%s)", strings.Join(f.labels, ","))
}

// Type conversion utilities.

func asFilter(impl fmt.Stringer) *Filter {
	return &Filter{
		Strategy: impl,
	}
}

func filterValue(value interface{}) (string, error) {
	// When value is a string, return it directly.
	if s, ok := value.(string); ok {
		return s, nil
	}
	// When value is a []interface{}, convert it into a []string.
	if s, ok := value.([]interface{}); ok {
		strslice := []string{}
		for _, v := range s {
			if str, ok := v.(string); ok {
				strslice = append(strslice, str)
			} else {
				return "", errors.Errorf("non-string \"%v\" in filter value", v)
			}
		}
		value = strslice
	}
	// When value is a []string, return the result of a strings.Join.
	if s, ok := value.([]string); ok {
		return strings.Join(s, ","), nil
	}
	// Anything else is an error.
	return "", errors.Errorf("invalid data type \"%#v\" for filter value", value)
}

// Mass filtering utilities.

// FilterIncludesIssues is a predicate to determine whether or not a set of
// filters should include issues.
func FilterIncludesIssues(filters []*Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Strategy.(IsFilter); ok && f.PullRequestOnly {
			return false
		}
	}
	return true
}

// FilterIncludesPullRequests is similar to FilterIncludesIssues only with
// respect to pull requests.
func FilterIncludesPullRequests(filters []*Filter) bool {
	for _, filter := range filters {
		if f, ok := filter.Strategy.(IsFilter); ok && !f.PullRequestOnly {
			return false
		}
	}
	return true
}
