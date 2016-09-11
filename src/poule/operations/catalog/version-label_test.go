package catalog

import (
	"testing"

	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
)

func makeContext() (*test.TestClient, *operations.Context) {
	clt := &test.TestClient{}
	return clt, &operations.Context{
		Client:     clt,
		Username:   test.Username,
		Repository: test.Repository,
	}
}

func TestVersionLabel(t *testing.T) {
	for expected, body := range map[string]string{
		"version/1.11":        "Body. Server: Version: 1.11.0. Trailing.",
		"version/1.12":        "Server:\n \tVersion:\t  1.12.1",
		"version/1.13":        "Server: Version: 1.13.1-rc1",
		"version/1.14":        "Server: Version: 1.14.1-cs2",
		"version/1.15":        "Server: Version: 1.15.3pouet",
		"version/master":      "Server: Version: 1.2.3-dev",
		"version/unsupported": "Server: Version: 1.2.3-toto",
	} {
		clt, ctx := makeContext()
		operation := versionLabelOperation{}

		issue := test.NewIssueBuilder(test.IssueNumber).Body(body).Item()
		clt.MockIssues.
			On("AddLabelsToIssue", ctx.Username, ctx.Repository, test.IssueNumber, []string{expected}).
			Return([]github.Label{github.Label{Name: test.MakeString(expected)}}, nil, nil)

		res, userData, err := operation.Filter(ctx, issue)
		if err != nil {
			t.Fatalf("Filter returned unexpected error %v", err)
		}
		if res != operations.Accept {
			t.Fatalf("Expected filter to accept %q, got %v", body, res)
		}
		if err := operation.Apply(ctx, issue, userData); err != nil {
			t.Fatalf("Expected no error from Apply(), got %v", err)
		}

		clt.MockIssues.AssertExpectations(t)
	}
}

func TestVersionLabelRejects(t *testing.T) {
	_, ctx := makeContext()
	operation := versionLabelOperation{}

	for _, body := range []string{
		"Body",
		"1.11.0",
		"Version: 1.12.0",
	} {
		issue := test.NewIssueBuilder(test.IssueNumber).Body(body).Item()
		if res, _, err := operation.Filter(ctx, issue); err != nil {
			t.Fatalf("Filter returned unexpected error %v", err)
		} else if res != operations.Reject {
			t.Fatalf("Filter returned unexpected result %v", res, body)
		}
	}
}
