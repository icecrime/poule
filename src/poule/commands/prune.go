package commands

import (
	"log"

	"poule/operations"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

var PruneCommand = cli.Command{
	Name:   "prune",
	Usage:  "Prune inactive or deprecated issues",
	Action: doPruneCommand,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "action",
			Usage: "Action to take on inactive issues (close, or warn)",
			Value: "warn",
		},
		cli.StringFlag{
			Name:  "grace-period",
			Usage: "Grace period to include in the warning message",
			Value: "2 weeks",
		},
	},
}

func doPruneCommand(c *cli.Context) {
	action := c.String("action")
	switch action {
	case "close", "warn":
		break
	default:
		log.Fatalf("Invalid value %q for action", action)
	}

	operations.RunIssueOperation(c, &pruneOperation{})
}

type pruneOperation struct{}

func (o *pruneOperation) Apply(c *operations.Context, issue *github.Issue, userData interface{}) error {
	return nil
}

func (o *pruneOperation) Describe(c *operations.Context, issue *github.Issue, userData interface{}) string {
	return ""
}

func (o *pruneOperation) Filter(c *operations.Context, issue *github.Issue) (bool, interface{}) {
	// Get issue comments.
	opts := github.IssueListCommentsOptions{
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
	_, _, err := c.Client.Issues.ListComments(c.Username, c.Repository, *issue.Number, &opts)
	if err != nil {
		log.Fatalf("Error listing comments for issue %d: %v", *issue.Number, err)
	}

	return false, nil
}

func (o *pruneOperation) ListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}
