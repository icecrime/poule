package test

import "testing"

func AssertExpectations(clt *TestClient, t *testing.T) {
	clt.MockIssues.AssertExpectations(t)
	clt.MockPullRequests.AssertExpectations(t)
	clt.MockRepositories.AssertExpectations(t)
}

func MakeInt(value int) *int {
	v := new(int)
	*v = value
	return v
}

func MakeString(value string) *string {
	v := new(string)
	*v = value
	return v
}
