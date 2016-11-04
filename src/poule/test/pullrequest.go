package test

import (
	"fmt"
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
	return gh.MakePullRequestItem(p.Value)
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

func (p *PullRequestBuilder) Body(body string) *PullRequestBuilder {
	p.Value.Body = MakeString(body)
	return p
}

func (p *PullRequestBuilder) Commits(commits int) *PullRequestBuilder {
	p.Value.Commits = MakeInt(commits)
	return p
}

func (p *PullRequestBuilder) HeadBranch(username, repository, SHA string) *PullRequestBuilder {
	p.Value.Head = &github.PullRequestBranch{
		Ref: MakeString(SHA),
		Repo: &github.Repository{
			FullName: MakeString(username + "/" + repository),
			Name:     MakeString(repository),
			Owner: &github.User{
				Login: MakeString(username),
			},
			SSHURL: MakeString(fmt.Sprintf("ssh@%s", repository)),
		},
		SHA: MakeString(SHA),
	}
	return p
}

func (p *PullRequestBuilder) Number(number int) *PullRequestBuilder {
	p.Value.Number = MakeInt(number)
	return p
}

func (p *PullRequestBuilder) Title(title string) *PullRequestBuilder {
	p.Value.Title = MakeString(title)
	return p
}
