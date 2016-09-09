package test

import (
	"poule/gh"

	"github.com/google/go-github/github"
)

func NewPullRequestBuilder(number int) *PullRequestBuilder {
	return &PullRequestBuilder{
		Value: &github.PullRequest{
			Number: MakeInt(number),
		},
	}
}

type PullRequestBuilder struct {
	Value *github.PullRequest
}

func (p *PullRequestBuilder) Item() gh.Item {
	return gh.MakeItem(p.Value)
}

func (p *PullRequestBuilder) BaseBranch(username, repository, SHA string) *PullRequestBuilder {
	p.Value.Base = &github.PullRequestBranch{
		Repo: &github.Repository{
			FullName: MakeString(username + "/" + repository),
			Name:     MakeString(repository),
			Owner: &github.User{
				Login: MakeString(username),
			},
		},
		SHA: MakeString(SHA),
	}
	return p
}

func (p *PullRequestBuilder) HeadBranch(username, repository, SHA string) *PullRequestBuilder {
	p.Value.Head = &github.PullRequestBranch{
		Repo: &github.Repository{
			FullName: MakeString(username + "/" + repository),
			Name:     MakeString(repository),
			Owner: &github.User{
				Login: MakeString(username),
			},
		},
		SHA: MakeString(SHA),
	}
	return p
}

func (p *PullRequestBuilder) Number(number int) *PullRequestBuilder {
	p.Value.Number = MakeInt(number)
	return p
}
