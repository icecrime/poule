package catalog

import (
	"testing"
	"time"

	"poule/operations"
	"poule/test"
	"poule/utils"

	"github.com/google/go-github/github"
)

func mockBuilder(pr *github.PullRequest, context string) error {
	return nil
}

func TestRebuild(t *testing.T) {
	gitSha := "d34db3333f"
	issue := &github.Issue{
		Number: makeInt(test.IssueNumber),
	}
	pullrequest := &github.PullRequest{
		Base: &github.PullRequestBranch{
			Repo: &github.Repository{
				FullName: makeString("fullName"),
				Name:     makeString(test.Repository),
				Owner: &github.User{
					Login: makeString(test.Username),
				},
			},
		},
		Head: &github.PullRequestBranch{
			SHA: makeString(gitSha),
		},
		Number: makeInt(test.IssueNumber),
	}

	//var m mock.Mock
	operation := prRebuild{
		Builder: func(pr *github.PullRequest, context string) error {
			return nil
			//return m.Called(pr, context).Error(0)
		},
		Configurations: []string{"test"},
	}

	now := time.Now()
	clt, ctx := makeContext()
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)
	clt.MockRepositories.
		On("ListStatuses", ctx.Username, ctx.Repository, gitSha, (*github.ListOptions)(nil)).
		Return([]github.RepoStatus{
			github.RepoStatus{
				Context:   makeString("test"),
				CreatedAt: &now,
				State:     makeString("failure"),
			},
		}, nil, nil)

	res, userData := operation.Filter(ctx, pullrequest)
	if res != operations.Accept {
		t.Fatalf("Rebuild filer should accept issue with failure")
	}
	if err := operation.Apply(ctx, pullrequest, userData); err != nil {
		t.Fatalf("Rebuild apply returned unexpected error %v", err)
	}

	clt.MockRepositories.AssertExpectations(t)
}

func TestRebuildSkipFailing(t *testing.T) {
	issue := &github.Issue{
		Number: makeInt(test.IssueNumber),
		Labels: []github.Label{
			github.Label{
				Name: makeString(utils.FailingCILabel),
			},
		},
	}
	pullrequest := &github.PullRequest{
		Base: &github.PullRequestBranch{
			Repo: &github.Repository{
				FullName: makeString("fullName"),
				Name:     makeString(test.Repository),
				Owner: &github.User{
					Login: makeString(test.Username),
				},
			},
		},
		Number: makeInt(test.IssueNumber),
	}

	clt, ctx := makeContext()
	clt.MockIssues.On("Get", ctx.Username, ctx.Repository, test.IssueNumber).Return(issue, nil, nil)

	operation := prRebuild{Configurations: []string{"test"}}
	if res, _ := operation.Filter(ctx, pullrequest); res != operations.Reject {
		t.Fatalf("Rebuild filter should reject issue with label %q", test.IssueNumber)
	}

	clt.MockIssues.AssertExpectations(t)
}
