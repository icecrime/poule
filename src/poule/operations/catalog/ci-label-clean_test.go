package catalog

import (
	"testing"
	"time"

	"github.com/google/go-github/github"

	"poule/configuration"
	"poule/gh"

	"poule/operations"
	"poule/test"
)

func TestCILabelClean(t *testing.T) {
	clt, ctx := makeContext()
	operation := &ciLabelCleanOperation{}

	// Create test pull request and related issue object.
	issue := test.NewIssueBuilder(test.IssueNumber).
		Labels([]string{configuration.FailingCILabel}).Value
	pullr := test.NewPullRequestBuilder(test.IssueNumber).
		HeadBranch(ctx.Username, ctx.Repository, "head", test.CommitSHA[0]).
		BaseBranch(ctx.Username, ctx.Repository, "base", test.CommitSHA[1]).Value
	clt.MockIssues.
		On("RemoveLabelForIssue", ctx.Username, ctx.Repository, test.IssueNumber, configuration.FailingCILabel).
		Return(nil, nil)
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	// Mock GitHub API replies to statuses retrieval.
	currentTime := time.Now()
	repoStatuses := []*github.RepoStatus{
		test.MakeStatus("conf_1", "success", currentTime.Add(-24*time.Hour)),
		test.MakeStatus("conf_2", "success", currentTime.Add(-1*time.Hour)),
		test.MakeStatus("conf_2", "pending", currentTime.Add(-2*time.Hour)),
		test.MakeStatus("conf_2", "failure", currentTime.Add(-3*time.Hour)),
	}
	clt.MockRepositories.On("ListStatuses", ctx.Username, ctx.Repository, test.CommitSHA[0], (*github.ListOptions)(nil)).Return(repoStatuses, nil, nil)

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
