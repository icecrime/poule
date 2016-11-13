package test

import "testing"

// AssertExpectations asserts mock expectations for all different GitHub services.
func AssertExpectations(clt *TestClient, t *testing.T) {
	clt.MockIssues.AssertExpectations(t)
	clt.MockPullRequests.AssertExpectations(t)
	clt.MockRepositories.AssertExpectations(t)
}
