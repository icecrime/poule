package catalog

import (
	"testing"

	"poule/operations"
	"poule/test"

	"github.com/google/go-github/github"
)

func TestLabel(t *testing.T) {
	clt, ctx := makeContext()
	item := test.NewPullRequestBuilder(test.IssueNumber).
		Title("This is the title of a pull request").
		Body("Lorem ipsum dolor sit amet, consectetur adipiscing elit").
		Item()

	// Create test pattern mapping.
	patterns := map[string]interface{}{}
	patterns["label1"] = []string{`not found`, `pull request`}
	patterns["label2"] = []string{`this is`, `incorrect`}
	patterns["label3"] = []string{`lorem\s+.psum`}
	patterns["label4"] = []string{`this is not found`}
	patterns["label5"] = []string{`This`}

	// Set up the mock objects.
	expected := []string{"label1", "label2", "label3"}
	clt.MockIssues.
		On("AddLabelsToIssue", ctx.Username, ctx.Repository, test.IssueNumber, expected).
		Return([]*github.Label{}, nil, nil)

	// Create the label operation with the test mapping.
	config := map[string]interface{}{"patterns": patterns}
	op, err := (&labelDescriptor{}).OperationFromConfig(config)
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
	test.AssertExpectations(clt, t)
}
