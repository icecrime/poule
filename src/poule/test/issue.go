package test

import (
	"poule/gh"

	"github.com/google/go-github/github"
)

// NewIssueBuilder returns a new IssueBuilder instance.
func NewIssueBuilder(number int) *IssueBuilder {
	return &IssueBuilder{
		Value: &github.Issue{
			Number: github.Int(number),
		},
	}
}

// IssueBuilder is a helper type to generate an issue object.
type IssueBuilder struct {
	Value *github.Issue
}

// Item returns the the underlying issue as an item.
func (p *IssueBuilder) Item() gh.Item {
	return gh.MakeIssueItem(p.Value)
}

// Body sets the body attribute of the issue.
func (p *IssueBuilder) Body(body string) *IssueBuilder {
	p.Value.Body = github.String(body)
	return p
}

// Labels sets the labels of the issue.
func (p *IssueBuilder) Labels(names []string) *IssueBuilder {
	for _, name := range names {
		p.Value.Labels = append(p.Value.Labels, MakeLabel(name))
	}
	return p
}

// Number sets the number of the issue.
func (p *IssueBuilder) Number(number int) *IssueBuilder {
	p.Value.Number = github.Int(number)
	return p
}
