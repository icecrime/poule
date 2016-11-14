package catalog

import (
	"errors"
	"fmt"
	"net/http"
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
		BaseBranch(ctx.Username, ctx.Repository, "base", "0x123").
		HeadBranch(ctx.Username, ctx.Repository, "head", "0x456").
		Commits(1).
		Item()

	// Set up the mock objects.
	clt.MockIssues.
		On("AddLabelsToIssue", ctx.Username, ctx.Repository, test.IssueNumber, []string{testDCOFailureLabel}).
		Return([]*github.Label{}, nil, nil)

	clt.MockPullRequests.
		On("ListCommits", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return([]*github.RepositoryCommit{
			{
				Commit: &github.Commit{
					SHA:     github.String(test.CommitSHA[0]),
					Message: github.String("Commit message"),
				},
			},
			{
				Commit: &github.Commit{
					SHA:     github.String(test.CommitSHA[1]),
					Message: github.String("Signed-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>"),
				},
			},
		}, nil, nil)

	clt.MockIssues.
		On("ListComments", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.IssueListCommentsOptions")).
		Return([]*github.IssueComment{}, &github.Response{NextPage: 0}, nil)

	clt.MockIssues.
		On("CreateComment", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.IssueComment")).
		Return(&github.IssueComment{}, nil, nil)

	clt.MockRepositories.
		On("CreateStatus", ctx.Username, ctx.Repository, "0x456", mock.AnythingOfType("*github.RepoStatus")).
		Run(func(args mock.Arguments) {
			arg := args.Get(3).(*github.RepoStatus)
			if expected := "failure"; arg.State == nil {
				t.Fatalf("Expected repoStatus to be %q, got <nil>", expected)
			} else if *arg.State != expected {
				t.Fatalf("Expected repoStatus to be %q, got %q", expected, *arg.State)
			}
		}).
		Return(nil, nil, nil)

	dcoTestStub(t, ctx, item)

	// Verify that the posted comment has the expected automated token.
	test.AssertExpectations(clt, t)
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
		BaseBranch(ctx.Username, ctx.Repository, "base", "0x123").
		HeadBranch(ctx.Username, ctx.Repository, "head", "0x456").
		Commits(1).
		Item()

	// Set up the mock objects.
	clt.MockIssues.
		On("RemoveLabelForIssue", ctx.Username, ctx.Repository, test.IssueNumber, testDCOFailureLabel).
		Return(nil, nil)

	clt.MockPullRequests.
		On("ListCommits", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return([]*github.RepositoryCommit{
			{
				SHA:     github.String(test.CommitSHA[0]),
				Message: github.String("This is signed.\nSigned-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>"),
			},
			{
				SHA:     github.String(test.CommitSHA[1]),
				Message: github.String("This too.\n\tSigned-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>  \nYep.\n"),
			},
		}, nil, nil)

	clt.MockIssues.
		On("ListComments", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.IssueListCommentsOptions")).
		Return([]*github.IssueComment{
			{
				ID:   github.Int(test.CommentID),
				Body: github.String(fmt.Sprintf("%s\nPlease sign your commit!", dcoCommentToken)),
			},
			{
				ID:   github.Int(test.CommentID + 1),
				Body: github.String("Merge it!"),
			},
			{
				ID:   github.Int(test.CommentID + 2),
				Body: github.String("Unrelated comment."),
			},
		}, &github.Response{NextPage: 0}, nil)

	clt.MockIssues.
		On("DeleteComment", ctx.Username, ctx.Repository, test.CommentID).
		Return(nil, nil)

	clt.MockRepositories.
		On("CreateStatus", ctx.Username, ctx.Repository, "0x456", mock.AnythingOfType("*github.RepoStatus")).
		Run(func(args mock.Arguments) {
			arg := args.Get(3).(*github.RepoStatus)
			if expected := "success"; arg.State == nil {
				t.Fatalf("Expected repoStatus to be %q, got <nil>", expected)
			} else if *arg.State != expected {
				t.Fatalf("Expected repoStatus to be %q, got %q", expected, *arg.State)
			}
		}).
		Return(nil, nil, nil)

	dcoTestStub(t, ctx, item)
}

func TestDCOSuccessLabelMissing(t *testing.T) {
	clt, ctx := makeContext()
	item := test.NewPullRequestBuilder(test.IssueNumber).
		Title("This is the title of a pull request").
		Body("Lorem ipsum dolor sit amet, consectetur adipiscing elit").
		BaseBranch(ctx.Username, ctx.Repository, "base", "0x123").
		HeadBranch(ctx.Username, ctx.Repository, "head", "0x456").
		Commits(1).
		Item()

	// Set up the mock objects.
	clt.MockIssues.
		On("RemoveLabelForIssue", ctx.Username, ctx.Repository, test.IssueNumber, testDCOFailureLabel).
		Return(&github.Response{
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
		}, errors.New("non-nil error"))

	clt.MockPullRequests.
		On("ListCommits", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.ListOptions")).
		Return([]*github.RepositoryCommit{
			{
				SHA:     github.String(test.CommitSHA[0]),
				Message: github.String("This is signed.\nSigned-off-by: Arnaud Porterie (icecrime) <arnaud.porterie@docker.com>"),
			},
		}, nil, nil)

	clt.MockIssues.
		On("ListComments", ctx.Username, ctx.Repository, test.IssueNumber, mock.AnythingOfType("*github.IssueListCommentsOptions")).
		Return([]*github.IssueComment{}, &github.Response{NextPage: 0}, nil)

	clt.MockRepositories.
		On("CreateStatus", ctx.Username, ctx.Repository, "0x456", mock.AnythingOfType("*github.RepoStatus")).
		Return(nil, nil, nil)

	dcoTestStub(t, ctx, item)
}
