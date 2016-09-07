package catalog

import (
	"testing"

	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
)

func makeInt(value int) *int {
	v := new(int)
	*v = value
	return v
}

func makeString(value string) *string {
	v := new(string)
	*v = value
	return v
}

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
		"version/master":      "Server: Version: 1.2.3-dev",
		"version/unsupported": "Server: Version: 1.2.3-toto",
	} {
		clt, ctx := makeContext()
		issue := &github.Issue{Body: &body, Number: makeInt(test.IssueNumber)}

		clt.MockIssues.
			On("AddLabelsToIssue", ctx.Username, ctx.Repository, test.IssueNumber, []string{expected}).
			Return([]github.Label{github.Label{Name: makeString(expected)}}, nil, nil)

		operation := versionLabel{}
		res, userData := operation.Filter(ctx, issue)
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
	operation := versionLabel{}
	for _, body := range []string{"Body", "1.11.0", "Version: 1.11.0"} {
		issue := &github.Issue{Body: &body, Number: makeInt(test.IssueNumber)}
		if res, _ := operation.Filter(ctx, issue); res != operations.Reject {
			t.Fatalf("Unexpected result %v when filtering %q", res, body)
		}
	}
}
