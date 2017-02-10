package catalog

import (
	"strings"

	"poule/operations"

	"github.com/google/go-github/github"
)

func deleteAutomatedComments(c *operations.Context, pr *github.PullRequest, substr string) error {
	automatedComments, err := findAutomatedComments(c, pr, substr)
	if err != nil {
		return err
	}
	for _, comment := range automatedComments {
		if _, err := c.Client.Issues().DeleteComment(c.Username, c.Repository, *comment.ID); err != nil {
			return err
		}
	}
	return nil
}

func findAutomatedComments(c *operations.Context, pr *github.PullRequest, substr string) ([]*github.IssueComment, error) {
	automatedComments := []*github.IssueComment{}
	issuesListOptions := &github.IssueListCommentsOptions{
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}

	// Retrieve all comments for that pull request.
	for page := 1; page != 0; {
		issuesListOptions.ListOptions.Page = page
		comments, resp, err := c.Client.Issues().ListComments(c.Username, c.Repository, *pr.Number, issuesListOptions)
		if err != nil {
			return nil, err
		}

		// Go through the comments looking for the automated token.
		// TODO Add a check that the author of the comment corresponds to the owner of the GitHub
		// token in use.
		for i := range comments {
			comment := comments[i]
			if comment.Body != nil && strings.Contains(*comment.Body, substr) {
				automatedComments = append(automatedComments, comment)
			}
		}

		page = resp.NextPage
	}
	return automatedComments, nil
}
