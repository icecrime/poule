package test

import "testing"

func AssertExpectations(clt *TestClient, t *testing.T) {
	clt.MockIssues.AssertExpectations(t)
	clt.MockPullRequests.AssertExpectations(t)
	clt.MockRepositories.AssertExpectations(t)
}

func MakeBool(value bool) *bool {
	v := new(bool)
	*v = value
	return v
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
