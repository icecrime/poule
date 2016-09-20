package catalog

import (
	"fmt"
	"strings"
	"testing"

	"poule/gh"
	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
)

const testDCOFailureLabel = "y-u-no-dco"

func dcoTestStub(t *testing.T, ctx *operations.Context, item gh.Item) {
	// Create the dco-check operation.
	config := operations.Configuration{"unsigned-label": testDCOFailureLabel}
	op, err := (&dcoCheckDescriptor{}).OperationFromConfig(config)
	if err != nil {
		t.Fatalf("OperationFromConfig returned unexpected error %v", err)
	}

	// Call into the operation.
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
}

func TestDCOFailure(t *testing.T) {
	clt, ctx := makeContext()
	item := test.NewPullRequestBuilder(test.IssueNumber).
		Title("This is the title of a pull request").
		Body("Lorem ipsum dolor sit amet, consectetur adipiscing elit").
		BaseBranch(ctx.Username, ctx.Repository, "0x123").
		HeadBranch(ctx.Username, ctx.Repository, "0x456").
		Commits(1).
		Item()

	// Set up the mock objects.
	clt.MockIssues.
		On("AddLabelsToIssue", ctx.Username, ctx.Repository, test.IssueNumber, []string{testDCOFailureLabel}).
		Return([]github.Label{}, nil, nil)

	clt.MockPullRequests.
		On("ListCommits", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return([]github.RepositoryCommit{
			github.RepositoryCommit{
				SHA:     test.MakeString(test.CommitSHA[0]),
				Message: test.MakeString("Commit message"),
			},
			github.RepositoryCommit{
				SHA:     test.MakeString(test.CommitSHA[1]),
				Message: test.MakeString("Signed-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>"),
			},
		}, nil, nil)

	clt.MockPullRequests.
		On("ListComments", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.PullRequestListCommentsOptions")).
		Return([]github.PullRequestComment{}, nil, nil)

	clt.MockPullRequests.
		On("CreateComment", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.PullRequestComment")).
		Return(&github.PullRequestComment{}, nil, nil)

	dcoTestStub(t, ctx, item)

	// Verify that the posted comment has the expected automated token.
	clt.MockIssues.AssertExpectations(t)
	for _, call := range clt.MockPullRequests.Calls {
		if call.Method == "CreateComment" {
			if comment := call.Arguments[3].(*github.PullRequestComment); !strings.Contains(*comment.Body, dcoCommentToken) {
				t.Fatalf("Automated comment doesn't contain the expected token %q", dcoCommentToken)
			}
			break
		}
	}
}

func TestDCOSuccess(t *testing.T) {
	clt, ctx := makeContext()
	item := test.NewPullRequestBuilder(test.IssueNumber).
		Title("This is the title of a pull request").
		Body("Lorem ipsum dolor sit amet, consectetur adipiscing elit").
		BaseBranch(ctx.Username, ctx.Repository, "0x123").
		HeadBranch(ctx.Username, ctx.Repository, "0x456").
		Commits(1).
		Item()

	// Set up the mock objects.
	clt.MockIssues.
		On("RemoveLabelForIssue", ctx.Username, ctx.Repository, test.IssueNumber, testDCOFailureLabel).
		Return(nil, nil)

	clt.MockPullRequests.
		On("ListCommits", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return([]github.RepositoryCommit{
			github.RepositoryCommit{
				SHA:     test.MakeString(test.CommitSHA[0]),
				Message: test.MakeString("This is signed.\nSigned-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>"),
			},
			github.RepositoryCommit{
				SHA:     test.MakeString(test.CommitSHA[1]),
				Message: test.MakeString("This too.\n\tSigned-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>  \nYep.\n"),
			},
		}, nil, nil)

	clt.MockPullRequests.
		On("ListComments", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.PullRequestListCommentsOptions")).
		Return([]github.PullRequestComment{
			github.PullRequestComment{
				ID:   test.MakeInt(test.CommentID),
				Body: test.MakeString(fmt.Sprintf("%s\nPlease sign your commit!", dcoCommentToken)),
			},
			github.PullRequestComment{
				ID:   test.MakeInt(test.CommentID + 1),
				Body: test.MakeString("Merge it!"),
			},
			github.PullRequestComment{
				ID:   test.MakeInt(test.CommentID + 2),
				Body: test.MakeString("Unrelated comment."),
			},
		}, nil, nil)

	clt.MockPullRequests.
		On("DeleteComment", ctx.Username, ctx.Repository, test.CommentID).
		Return(nil, nil)

	dcoTestStub(t, ctx, item)
}
