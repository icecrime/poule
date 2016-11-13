package gh

import (
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

// Item is a union type that can encapsulate either a github.Issue or a
// github.PullRequest. This allows to have a single Operation interface and let
// the implementation handle according to its capabilities.
type Item struct {
	Issue       *github.Issue
	PullRequest *github.PullRequest
}

// MakeIssueItem create an Item wrapper around a GitHub issue.
func MakeIssueItem(issue *github.Issue) Item {
	return Item{
		Issue: issue,
	}
}

// MakePullRequestItem create an Item wrapper around a GitHub pull request.
func MakePullRequestItem(pullRequest *github.PullRequest) Item {
	return Item{
		PullRequest: pullRequest,
	}
}

// IsNil returns true when the item is not initialized.
func (i Item) IsNil() bool {
	return i.Issue == nil && i.PullRequest == nil
}

// IsIssue returns whether the item is strictly a GitHub issue (i.e., not the
// issue object of a corresponding pull request).
func (i Item) IsIssue() bool {
	// The `Issue` field can be non-nil even in the case of a pull request, as
	// we may have fetched the related issue object (for example to retrieve
	// labels).
	return i.PullRequest == nil
}

// IsPullRequest returns whether the item is a GitHub pull request.
func (i Item) IsPullRequest() bool {
	return i.PullRequest != nil
}

// Body returns the text body of the item.
func (i Item) Body() string {
	switch {
	case i.Issue != nil:
		return *i.Issue.Body
	case i.PullRequest != nil:
		return *i.PullRequest.Body
	default:
		panic("uninitialized item")
	}
}

// Number returns the number of the item.
func (i *Item) Number() int {
	switch {
	case i.Issue != nil:
		return *i.Issue.Number
	case i.PullRequest != nil:
		return *i.PullRequest.Number
	default:
		panic("uninitialized item")
	}
}

// Repository returns the repository full name of the item. In the case of a
// pull request, this is the destination repository.
func (i *Item) Repository() string {
	switch {
	case i.Issue != nil:
		return *i.Issue.Repository.FullName
	case i.PullRequest != nil:
		return *i.PullRequest.Base.Repo.FullName
	default:
		panic("uninitialized item")
	}
}

// Title returns the title of the item.
func (i *Item) Title() string {
	switch {
	case i.Issue != nil:
		return *i.Issue.Title
	case i.PullRequest != nil:
		return *i.PullRequest.Title
	default:
		panic("uninitialized item")
	}
}

// Type returns a string representation of the GitHub item type.
func (i *Item) Type() string {
	switch {
	case i.Issue != nil:
		return "issue"
	case i.PullRequest != nil:
		return "pull_request"
	default:
		return "<none>"
	}
}

// GetRelatedIssue retrieves and return the GitHub issue related to a pull
// request. This function will fail when called on a GitHub issue.
func (i *Item) GetRelatedIssue(client Client) (*github.Issue, error) {
	if i.Issue != nil {
		return i.Issue, nil
	} else if i.IsIssue() {
		return nil, errors.Errorf("GetRelatedIssue called on an issue")
	}

	issue, _, err := client.Issues().Get(
		*i.PullRequest.Base.Repo.Owner.Login,
		*i.PullRequest.Base.Repo.Name,
		*i.PullRequest.Number,
	)
	if err != nil {
		return nil, err
	}
	i.Issue = issue
	return i.Issue, nil
}
