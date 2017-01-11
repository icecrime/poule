package catalog

import (
	"testing"

	"poule/operations"
	"poule/test"
)

func TestAssign(t *testing.T) {
	clt, ctx := makeContext()

	// Create test pattern mapping.
	users := []string{"user1", "user2"}

	item := test.NewPullRequestBuilder(test.IssueNumber).
		Title("This is the title of a pull request").
		Body("Lorem ipsum dolor sit amet, consectetur adipiscing elit").
		UserLogin(users[0]).
		Item()

	// Set up the mock objects.
	expected := []string{"user2"}
	clt.MockIssues.
		On("AddAssignees", ctx.Username, ctx.Repository, test.IssueNumber, expected).
		Return(nil, nil, nil)

	// Create the assign operation with the test mapping.
	config := map[string]interface{}{"users": users}
	op, err := (&assignDescriptor{}).OperationFromConfig(config)
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
