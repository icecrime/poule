package test

import (
	"fmt"
	"poule/gh"

	"github.com/google/go-github/github"
)

func NewPullRequestBuilder(number int) *PullRequestBuilder {
	return &PullRequestBuilder{
		Value: &github.PullRequest{
			Number: github.Int(number),
		},
	}
}

type PullRequestBuilder struct {
	Value *github.PullRequest
}

func (p *PullRequestBuilder) Item() gh.Item {
	return gh.MakePullRequestItem(p.Value)
}

func (p *PullRequestBuilder) BaseBranch(username, repository, ref string, SHA string) *PullRequestBuilder {
	p.Value.Base = &github.PullRequestBranch{
		Ref: github.String(ref),
		Repo: &github.Repository{
			FullName: github.String(username + "/" + repository),
			Name:     github.String(repository),
			Owner: &github.User{
				Login: github.String(username),
			},
		},
		SHA: github.String(SHA),
	}
	return p
}

func (p *PullRequestBuilder) Body(body string) *PullRequestBuilder {
	p.Value.Body = github.String(body)
	return p
}

func (p *PullRequestBuilder) Commits(commits int) *PullRequestBuilder {
	p.Value.Commits = github.Int(commits)
	return p
}

func (p *PullRequestBuilder) HeadBranch(username, repository, ref string, SHA string) *PullRequestBuilder {
	p.Value.Head = &github.PullRequestBranch{
		Ref: github.String(ref),
		Repo: &github.Repository{
			FullName: github.String(username + "/" + repository),
			Name:     github.String(repository),
			Owner: &github.User{
				Login: github.String(username),
			},
			SSHURL: github.String(fmt.Sprintf("ssh@%s", repository)),
		},
		SHA: github.String(SHA),
	}
	return p
}

func (p *PullRequestBuilder) Merged(merged bool) *PullRequestBuilder {
	p.Value.Merged = github.Bool(merged)
	return p
}

func (p *PullRequestBuilder) Number(number int) *PullRequestBuilder {
	p.Value.Number = github.Int(number)
	return p
}

func (p *PullRequestBuilder) State(state string) *PullRequestBuilder {
	p.Value.State = github.String(state)
	return p
}

func (p *PullRequestBuilder) Title(title string) *PullRequestBuilder {
	p.Value.Title = github.String(title)
	return p
}
