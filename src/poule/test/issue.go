package test

import "github.com/google/go-github/github"

func NewIssueBuilder(number int) *IssueBuilder {
	return &IssueBuilder{
		Value: &github.Issue{
			Number: MakeInt(number),
		},
	}
}

type IssueBuilder struct {
	Value *github.Issue
}

func (p *IssueBuilder) Body(body string) *IssueBuilder {
	p.Value.Body = MakeString(body)
	return p
}

func (p *IssueBuilder) Labels(names []string) *IssueBuilder {
	for _, name := range names {
		p.Value.Labels = append(p.Value.Labels, MakeLabel(name))
	}
	return p
}

func (p *IssueBuilder) Number(number int) *IssueBuilder {
	p.Value.Number = MakeInt(number)
	return p
}
