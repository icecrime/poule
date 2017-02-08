package catalog

import (
	"fmt"
	"poule/operations"
	"strings"
	"time"

	"poule/configuration"
	"poule/gh"
	"poule/operations/settings"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&pruneDescriptor{})
}

type pruneDescriptor struct{}

type pruneConfig struct {
	Action            string `mapstructure:"action"`
	GracePeriod       string `mapstructure:"grace-period"`
	OutdatedThreshold string `mapstructure:"outdated-threshold"`
}

func (d *pruneDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "prune",
		Description: "Prune outdated issues",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "action",
				Usage: "action to take for outdated issues",
				Value: "ping",
			},
			cli.StringFlag{
				Name:  "grace-period",
				Usage: "grace period before closing",
				Value: "2w",
			},
			cli.StringFlag{
				Name:  "threshold",
				Usage: "threshold in days, weeks, months, or years",
				Value: "6m",
			},
		},
	}
}

func (d *pruneDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	pruneConfig := &pruneConfig{
		Action:            c.String("action"),
		GracePeriod:       c.String("grace-period"),
		OutdatedThreshold: c.String("threshold"),
	}
	return d.makeOperation(pruneConfig)
}

func (d *pruneDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	pruneConfig := &pruneConfig{}
	if err := mapstructure.Decode(c, &pruneConfig); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	return d.makeOperation(pruneConfig)
}

func (d *pruneDescriptor) makeOperation(config *pruneConfig) (operations.Operation, error) {
	var (
		err       error
		operation pruneOperation
	)
	if operation.action, err = parseAction(config.Action); err != nil {
		return nil, err
	}
	if operation.gracePeriod, err = settings.ParseExtDuration(config.GracePeriod); err != nil {
		return nil, err
	}
	if operation.outdatedThreshold, err = settings.ParseExtDuration(config.OutdatedThreshold); err != nil {
		return nil, err
	}
	return &operation, nil
}

type pruneOperation struct {
	action            string
	gracePeriod       settings.ExtDuration
	outdatedThreshold settings.ExtDuration
}

func (o *pruneOperation) Accepts() operations.AcceptedType {
	return operations.Issues
}

func (o *pruneOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	issue := item.Issue
	switch o.action {
	case "close":
		// TODO Find the last ping/warn message, and take the grace period into account.
		break
	case "force-close":
		state := "closed"
		_, _, err := c.Client.Issues().Edit(c.Username, c.Repository, *issue.Number, &github.IssueRequest{
			State: &state,
		})
		return err
	case "ping":
		body := formatPingComment(issue, o)
		_, _, err := c.Client.Issues().CreateComment(c.Username, c.Repository, *issue.Number, &github.IssueComment{
			Body: &body,
		})
		return err
	case "warn":
		body := formatWarnComment(issue, o)
		_, _, err := c.Client.Issues().CreateComment(c.Username, c.Repository, *issue.Number, &github.IssueComment{
			Body: &body,
		})
		return err
	}
	return nil
}

func (o *pruneOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	issue := item.Issue
	return fmt.Sprintf("Execute %s action on issue #%d (last commented on %s)",
		o.action, *issue.Number, userData.(time.Time).Format(time.RFC3339))
}

func (o *pruneOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Retrieve comments for that issue since our threshold plus our grace
	// period plus one day.
	issue := item.Issue
	comments, _, err := c.Client.Issues().ListComments(c.Username, c.Repository, *issue.Number, &github.IssueListCommentsOptions{
		Since: time.Now().Add(-1*o.outdatedThreshold.Duration()).Add(-1*o.gracePeriod.Duration()).AddDate(0, 0, -1),
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	})
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve comments for issue #%d", *issue.Number)
	}

	// Figure out the last time the issue was commented on.
	lastCommented := *issue.UpdatedAt
	for size := len(comments); size > 0; size-- {
		// Skip all comments produced by the tool itself (as indicated by the
		// presence of the PouleToken).
		if strings.Contains(*comments[size-1].Body, configuration.PouleToken) {
			comments = comments[0 : size-1]
			continue
		}
		lastCommented = *comments[size-1].UpdatedAt
		break
	}

	// Filter out issues which last commented date is under our threshold. We
	// retrieve the issues in ascending update order: no more issues will be
	// accepted after that.
	if !lastCommented.Add(o.outdatedThreshold.Duration()).Before(time.Now()) {
		return operations.Terminal, nil, nil
	}
	return operations.Accept, lastCommented, nil
}

func (o *pruneOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "asc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func (o *pruneOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	// pruneOperation doesn't apply to GitHub pull requests.
	return nil
}

func formatPingComment(issue *github.Issue, o *pruneOperation) string {
	comment := `<!-- %s:%s:%d%c -->
@%s It has been detected that this issue has not received any activity in over %s. Can you please let us know if it is still relevant:

- For a bug: do you still experience the issue with the latest version?
- For a feature request: was your request appropriately answered in a later version?

Thank you!`
	return fmt.Sprintf(comment,
		configuration.PouleToken,
		o.action,
		o.outdatedThreshold.Quantity,
		o.outdatedThreshold.Unit,
		*issue.User.Login,
		o.outdatedThreshold.String(),
	)
}

func formatWarnComment(issue *github.Issue, o *pruneOperation) string {
	comment := `%s
This issue will be **automatically closed in %s** unless it is commented on.
`
	base := formatPingComment(issue, o)
	return fmt.Sprintf(comment, base, o.gracePeriod.String())
}

func parseAction(action string) (string, error) {
	switch action {
	case "close", "force-close", "ping", "warn":
		break
	default:
		return "", fmt.Errorf("Invalid action %q", action)
	}
	return action, nil
}
