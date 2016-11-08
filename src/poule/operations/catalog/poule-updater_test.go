package catalog

import (
	"testing"

	"poule/configuration"
	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
)

type mockUpdater struct {
	mock.Mock
}

func (m *mockUpdater) UpdateCallback(repository string) error {
	return m.Called(repository).Error(0)
}

func TestPouleUpdater(t *testing.T) {
	clt, ctx := makeContext()

	// Create test pull request and related issue object.
	item := test.NewPullRequestBuilder(test.IssueNumber).
		Merged(true).
		HeadBranch(ctx.Username, ctx.Repository, "head", test.CommitSHA[0]).
		BaseBranch(ctx.Username, ctx.Repository, "master", test.CommitSHA[1]).Item()

	// Create a mock for the poule update callback.
	m := &mockUpdater{}
	m.On("UpdateCallback", item.Repository()).Return(nil)
	PouleUpdateCallback = m.UpdateCallback

	// Set up the mock objects.
	commitFiles := []*github.CommitFile{
		&github.CommitFile{
			Filename: github.String("Dockerfile"),
		},
		&github.CommitFile{
			Filename: github.String(configuration.PouleConfigurationFile),
		},
	}
	clt.MockPullRequests.
		On("ListFiles", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return(commitFiles, nil, nil)

	// Call into the operation.
	op := &pouleUpdaterOperation{}
	res, userData, err := op.Filter(ctx, item)
	if err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	}
	if res != operations.Accept {
		t.Fatalf("Filter returned unexpected result %v", res)
	}
	if err := op.Apply(ctx, item, userData); err != nil {
		t.Fatalf("Apply returned unexpected error %v", err)
	}
	m.AssertExpectations(t)
	test.AssertExpectations(clt, t)
}
