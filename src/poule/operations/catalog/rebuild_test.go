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

func makeRebuildOperation(configurations []string, statuses []string, label string) (operations.Operation, *mock.Mock) {
	m := &mockBuilder{}
	operation := &prRebuildOperation{
		Builder:        m.Rebuild,
		Configurations: configurations,
		Label:          label,
		Statuses:       statuses,
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
	}, []string{
		"error",
		"failure",
	}, "")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		State("open").
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to issue and statuses retrieval. We expect that
	// the operation will only attempt to rebuild the "conf_error" and the
	// "conf_fail" jobs, and neither the pending, the succesful one (even if it
	// did previously fail), and the one that failed but was not included in
	// the configurations to rebuild.
	currentTime := time.Now()
	repoStatuses := []*github.RepoStatus{
		test.MakeStatus("conf_error", "error", currentTime.Add(-24*time.Hour)),
		test.MakeStatus("conf_fail", "success", currentTime.Add(-1*time.Hour)),
		test.MakeStatus("conf_fail", "failure", currentTime),
		test.MakeStatus("conf_pending", "pending", currentTime),
		test.MakeStatus("conf_success", "failure", currentTime.Add(-3*time.Hour)),
		test.MakeStatus("conf_success", "success", currentTime.Add(-2*time.Hour)),
		test.MakeStatus("conf_other_error", "error", currentTime.Add(-25*time.Hour)),
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
	test.AssertExpectations(clt, t)
}

func TestRebuildAllConfigurations(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, mockBuilder := makeRebuildOperation([]string{}, []string{"success"}, "")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		State("open").
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to issue and statuses retrieval. We expect that
	// the operation will only attempt to rebuild the "conf_error" and the
	// "conf_fail" jobs, and neither the pending, the succesful one (even if it
	// did previously fail), and the one that failed but was not included in
	// the configurations to rebuild.
	currentTime := time.Now()
	repoStatuses := []*github.RepoStatus{
		test.MakeStatus("random_name", "success", currentTime.Add(-3*time.Hour)),
		test.MakeStatus("other_random_name", "failure", currentTime),
	}
	mockBuilder.On("Rebuild", pullr, "random_name").Return(nil)
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
	test.AssertExpectations(clt, t)
}

func TestRebuildSkipFailing(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, _ := makeRebuildOperation([]string{"test"}, []string{}, "")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Labels([]string{configuration.FailingCILabel}).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Call into the operation.
	item := gh.MakePullRequestItem(pullr)
	if res, _, err := operation.Filter(ctx, item); err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	} else if res != operations.Reject {
		t.Fatalf("Filter should reject issue with label %q", test.IssueNumber)
	}
	test.AssertExpectations(clt, t)
}

func TestRebuildExcludeClosed(t *testing.T) {
	clt, ctx := makeContext()
	operation, _ := makeRebuildOperation([]string{}, []string{"failure"}, "")

	// Create test pull request and related issue object.
	pullr := test.NewPullRequestBuilder(test.IssueNumber).State("closed").Value

	// Call into the operation.
	item := gh.MakePullRequestItem(pullr)
	if res, _, err := operation.Filter(ctx, item); err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	} else if res != operations.Reject {
		t.Fatalf("Filter returned unexpected result %v", res)
	}
	test.AssertExpectations(clt, t)
}

func TestRebuildWithLabelCriteria(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, mockBuilder := makeRebuildOperation([]string{"configuration"}, []string{"failure"}, "rebuild")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Labels([]string{"rebuild"}).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		State("open").
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to issue and statuses retrieval. We expect that
	// the operation will only attempt to rebuild the "conf_error" and the
	// "conf_fail" jobs, and neither the pending, the succesful one (even if it
	// did previously fail), and the one that failed but was not included in
	// the configurations to rebuild.
	currentTime := time.Now()
	repoStatuses := []*github.RepoStatus{
		test.MakeStatus("configuration", "failure", currentTime.Add(-1*time.Hour)),
	}
	mockBuilder.On("Rebuild", pullr, "configuration").Return(nil)
	clt.MockIssues.On("RemoveLabelForIssue", ctx.Username, ctx.Repository, test.IssueNumber, "rebuild").Return(nil, nil)
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
	test.AssertExpectations(clt, t)
}

func TestRebuildWithLabelCriteriaMissing(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, _ := makeRebuildOperation([]string{"configuration"}, []string{"failure"}, "rebuild")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		State("open").
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Call into the operation.
	item := gh.MakePullRequestItem(pullr)
	if res, _, err := operation.Filter(ctx, item); err != nil {
		t.Fatalf("Filter returned unexpected error %v", err)
	} else if res != operations.Reject {
		t.Fatalf("Filter returned unexpected result %v", res)
	}
	test.AssertExpectations(clt, t)
}

func TestRebuildNewConfiguration(t *testing.T) {
	clt, ctx := makeContext()
	commitSHA := "baddcafe"
	operation, mockBuilder := makeRebuildOperation([]string{"new"}, []string{"failure"}, "")

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).Labels([]string{"rebuild"}).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		State("open").
		HeadBranch(ctx.Username, ctx.Repository, "head", commitSHA).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[0]).Value
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to issue and statuses retrieval. We expect that
	// the operation will only attempt to rebuild the "conf_error" and the
	// "conf_fail" jobs, and neither the pending, the succesful one (even if it
	// did previously fail), and the one that failed but was not included in
	// the configurations to rebuild.
	currentTime := time.Now()
	repoStatuses := []*github.RepoStatus{
		test.MakeStatus("old", "failure", currentTime.Add(-1*time.Hour)),
	}
	mockBuilder.On("Rebuild", pullr, "new").Return(nil)
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
	test.AssertExpectations(clt, t)
}
