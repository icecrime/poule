package test

import (
	"poule/gh"
	"poule/test/mocks"
	"time"

	"github.com/google/go-github/github"
)

type TestClient struct {
	MockIssues       mocks.IssuesService
	MockPullRequests mocks.PullRequestsService
	MockRepositories mocks.RepositoriesService
}

func (t *TestClient) Issues() gh.IssuesService {
	return &t.MockIssues
}

func (t *TestClient) PullRequests() gh.PullRequestsService {
	return &t.MockPullRequests
}

func (t *TestClient) Repositories() gh.RepositoriesService {
	return &t.MockRepositories
}

func MakeLabel(name string) github.Label {
	return github.Label{
		Name: github.String(name),
	}
}

func MakeStatus(context, status string, createdAt time.Time) *github.RepoStatus {
	return &github.RepoStatus{
		Context:   &context,
		CreatedAt: &createdAt,
		State:     &status,
	}
}
