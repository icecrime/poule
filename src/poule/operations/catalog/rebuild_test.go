package catalog

import (
	"testing"
	"time"

	"poule/configuration"
	"poule/gh"
	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
)

type mockBuilder struct {
	mock.Mock
}

func (m *mockBuilder) Rebuild(pr *github.PullRequest, context string) error {
	return m.Called(pr, context).Error(0)
}

func makeRebuildOperation(configurations []string) (operations.Operation, *mock.Mock) {
	m := &mockBuilder{}
	operation := &prRebuildOperation{
		Builder:        m.Rebuild,
		Configurations: configurations,
	}
	return operation, &m.Mock
}

func TestRebuild(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, mockBuilder := makeRebuildOperation([]string{
		"conf_error",
		"conf_fail",
		"conf_pending",
		"conf_success",
	})

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		HeadBranch(ctx.Username, ctx.Repository, commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, test.CommitSHA).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to issue and statuses retrieval. We expect that
	// the operation will only attempt to rebuild the "conf_error" and the
	// "conf_fail" jobs, and neither the pending, the succesful one (even if it
	// did previously fail), and the one that failed but was not included in
	// the configurations to rebuild.
	currentTime := time.Now()
	repoStatuses := []github.RepoStatus{
		test.MakeStatus("conf_error", "error", currentTime.Add(-24*time.Hour)),
		test.MakeStatus("conf_fail", "success", currentTime.Add(-1*time.Hour)),
		test.MakeStatus("conf_fail", "failure", currentTime),
		test.MakeStatus("conf_pending", "pending", currentTime),
		test.MakeStatus("conf_success", "failure", currentTime.Add(-3*time.Hour)),
		test.MakeStatus("conf_success", "success", currentTime.Add(-2*time.Hour)),
		test.MakeStatus("conf_other_fail", "failure", currentTime),
	}
	mockBuilder.On("Rebuild", pullr, "conf_fail").Return(nil)
	mockBuilder.On("Rebuild", pullr, "conf_error").Return(nil)
	clt.MockRepositories.On("ListStatuses", ctx.Username, ctx.Repository, commitSHA, (*github.ListOptions)(nil)).Return(repoStatuses, nil, nil)

	// Call into the operation.
	item := gh.MakePullRequestItem(pullr)
	res, userData, err := operation.Filter(ctx, item)
	if err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	}
	if res != operations.Accept {
		t.Fatalf("Filter returned unexpected result %v", res)
	}
	if err := operation.Apply(ctx, item, userData); err != nil {
		t.Fatalf("Apply returned unexpected error %v", err)
	}
	clt.MockIssues.AssertExpectations(t)
	clt.MockRepositories.AssertExpectations(t)
}

func TestRebuildSkipFailing(t *testing.T) {
	clt, ctx := makeContext()
	operation, _ := makeRebuildOperation([]string{"test"})

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Labels([]string{configuration.FailingCILabel}).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).BaseBranch(ctx.Username, ctx.Repository, test.CommitSHA).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Call into the operation.
	item := gh.MakePullRequestItem(pullr)
	if res, _, err := operation.Filter(ctx, item); err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	} else if res != operations.Reject {
		t.Fatalf("Filter should reject issue with label %q", test.IssueNumber)
	}
	clt.MockIssues.AssertExpectations(t)
}
