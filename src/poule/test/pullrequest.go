package test

import (
	"fmt"
	"poule/gh"

	"github.com/google/go-github/github"
)

// NewPullRequestBuilder returns a new PullRequestBuilder instance.
func NewPullRequestBuilder(number int) *PullRequestBuilder {
	return &PullRequestBuilder{
		Value: &github.PullRequest{
			Number: github.Int(number),
		},
	}
}

// PullRequestBuilder is a helper type to generate a pull request object.
type PullRequestBuilder struct {
	Value *github.PullRequest
}

// Item returns the the underlying pull request as an item.
func (p *PullRequestBuilder) Item() gh.Item {
	return gh.MakePullRequestItem(p.Value)
}

// BasBranch sets the Base attribute of the pull request.
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

// Body sets the body attribute of the pull request.
func (p *PullRequestBuilder) Body(body string) *PullRequestBuilder {
	p.Value.Body = github.String(body)
	return p
}

// Commits sets the commits objects of the pull request.
func (p *PullRequestBuilder) Commits(commits int) *PullRequestBuilder {
	p.Value.Commits = github.Int(commits)
	return p
}

// HeadBranch sets the Head attribute of the pull request.
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

// Merged sets the Merge attribute of the pull request.
func (p *PullRequestBuilder) Merged(merged bool) *PullRequestBuilder {
	p.Value.Merged = github.Bool(merged)
	return p
}

// Number sets the number of the pull request.
func (p *PullRequestBuilder) Number(number int) *PullRequestBuilder {
	p.Value.Number = github.Int(number)
	return p
}

// State sets the state of the pull request.
func (p *PullRequestBuilder) State(state string) *PullRequestBuilder {
	p.Value.State = github.String(state)
	return p
}

// Title sets the title of the pull request.
func (p *PullRequestBuilder) Title(title string) *PullRequestBuilder {
	p.Value.Title = github.String(title)
	return p
}
