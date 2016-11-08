package test

import "testing"

func AssertExpectations(clt *TestClient, t *testing.T) {
	clt.MockIssues.AssertExpectations(t)
	clt.MockPullRequests.AssertExpectations(t)
	clt.MockRepositories.AssertExpectations(t)
}
