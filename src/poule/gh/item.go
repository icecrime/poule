package gh

import "github.com/google/go-github/github"

// Item is a union type that can encapsulate either a github.Issue or a
// github.PullRequest. This allows to have a single Operation interface and let
// the implementation handle according to its capabilities.
type Item struct {
	item interface{}
}

func MakeItem(item interface{}) Item {
	return Item{
		item: item,
	}
}

func (i *Item) IsIssue() bool {
	_, ok := i.item.(*github.Issue)
	return ok
}

func (i *Item) Issue() *github.Issue {
	return i.item.(*github.Issue)
}

func (i *Item) IsPullRequest() bool {
	_, ok := i.item.(*github.PullRequest)
	return ok
}

func (i *Item) PullRequest() *github.PullRequest {
	return i.item.(*github.PullRequest)
}
