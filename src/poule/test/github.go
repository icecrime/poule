package test

import (
	"poule/gh"
	"poule/test/mocks"
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
