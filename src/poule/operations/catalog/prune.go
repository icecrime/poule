package catalog

import (
	"fmt"
	"log"
	"os"
	"poule/operations"
	"strings"
	"time"

	"poule/utils"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&pruneDescriptor{})
}

func doRunPrune(c *cli.Context) {
	/*
		action, err := parseAction(c.String("action"))
		if err != nil {
			log.Fatal(err)
		}

		filters, err := parseFilters(c.StringSlice("filter"))
		if err != nil {
			log.Fatal(err)
		}

		gracePeriod, err := parseExtDuration(c.String("grace-period"))
		if err != nil {
			log.Fatal(err)
		}

		outdatedThreshold, err := parseExtDuration(c.String("threshold"))
		if err != nil {
			log.Fatal(err)
		}
	*/

	/*
		operations.RunIssueOperation(c, &prune{
			action:            action,
			filters:           filters,
			gracePeriod:       gracePeriod,
			outdatedThreshold: outdatedThreshold,
		})
	*/
}

type pruneDescriptor struct{}

func (d *pruneDescriptor) Description() string {
	return "prune outdated issues"
}

func (d *pruneDescriptor) Name() string {
	return "prune"
}

func (d *pruneDescriptor) Command() cli.Command {
	return cli.Command{
		Name:  d.Name(),
		Usage: "prune outdated issues",
		Action: func(c *cli.Context) {
			doRunPrune(c)
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "action",
				Usage: "action to take for outdated issues (\"ping\", \"warn\", \"close\", or \"force-close\")",
				Value: "ping",
			},
			cli.StringSliceFlag{
				Name:  "filter",
				Usage: "filter based on issue attributes",
			},
			cli.StringFlag{
				Name:  "grace-period",
				Usage: "grace period before closing",
				Value: "2w",
			},
			cli.StringFlag{
				Name:  "threshold",
				Usage: "threshold in days, weeks, months, or years (e.g., \"4d\", \"3w\", \"2m\", or \"1y\"",
				Value: "6m",
			},
		},
	}
}

func (d *pruneDescriptor) OperationFromCli(c *cli.Context) Operation {
	return &prune{}
}

func (d *pruneDescriptor) OperationFromConfig(c operations.Configuration) Operation {
	return &prune{}
}

type prune struct {
	action            string
	filters           []utils.IssueFilter
	gracePeriod       extDuration
	outdatedThreshold extDuration
}

var i int

func (o *prune) Apply(c *operations.Context, issue *github.Issue, userData interface{}) error {
	switch o.action {
	case "close":
		// TODO Find the last ping/warn message, and take the grace period into account.
		break
	case "force-close":
		state := "closed"
		_, _, err := c.Client.Issues.Edit(c.Username, c.Repository, *issue.Number, &github.IssueRequest{
			State: &state,
		})
		return err
	case "ping":
		body := formatPingComment(issue, o)
		_, _, err := c.Client.Issues.CreateComment(c.Username, c.Repository, *issue.Number, &github.IssueComment{
			Body: &body,
		})
		return err
	case "warn":
		body := formatWarnComment(issue, o)
		_, _, err := c.Client.Issues.CreateComment(c.Username, c.Repository, *issue.Number, &github.IssueComment{
			Body: &body,
		})
		return err
	}
	return nil
}

func (o *prune) Describe(c *operations.Context, issue *github.Issue, userData interface{}) string {
	i++
	if i > 30 {
		os.Exit(0)
	}
	return fmt.Sprintf("Execute %s action on issue #%d (last commented on %s)",
		o.action, *issue.Number, userData.(time.Time).Format(time.RFC3339))
}

func (o *prune) Filter(c *operations.Context, issue *github.Issue) (operations.FilterResult, interface{}) {
	// Apply filters, if any.
	for _, filter := range o.filters {
		if !filter.Apply(issue) {
			return operations.Reject, nil
		}
	}

	// Retrieve comments for that issue since our threshold plus our grace
	// period plus one day.
	comments, _, err := c.Client.Issues.ListComments(c.Username, c.Repository, *issue.Number, &github.IssueListCommentsOptions{
		Since: time.Now().Add(-1*o.outdatedThreshold.Duration()).Add(-1*o.gracePeriod.Duration()).AddDate(0, 0, -1),
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	})
	if err != nil {
		log.Fatalf("Error getting comments for issue %d: %v", *issue.Number, err)
	}

	// Figure out the last time the issue was commented on.
	lastCommented := *issue.UpdatedAt
	for size := len(comments); size > 0; size-- {
		// Skip all comments produced by the tool itself (as indicated by the
		// presence of the PouleToken).
		if strings.Contains(*comments[size-1].Body, utils.PouleToken) {
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
		return operations.Terminal, nil
	}
	return operations.Accept, lastCommented
}

func (o *prune) ListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "asc",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func formatPingComment(issue *github.Issue, o *prune) string {
	comment := `<!-- %s:%s:%d%c -->
@%s It has been detected that this issue has not received any activity in over %s. Can you please let us know if it is still relevant:

- For a bug: do you still experience the issue with the latest version?
- For a feature request: was your request appropriately answered in a later version?

Thank you!`
	return fmt.Sprintf(comment,
		utils.PouleToken,
		o.action,
		o.outdatedThreshold.quantity,
		o.outdatedThreshold.unit,
		*issue.User.Login,
		o.outdatedThreshold.String(),
	)
}

func formatWarnComment(issue *github.Issue, o *prune) string {
	comment := `%s
Thank you very much for your help! The issue will be **automatically closed in %s** unless it is commented on.
`
	base := formatPingComment(issue, o)
	return fmt.Sprintf(comment, base, o.gracePeriod.String())
}

type extDuration struct {
	quantity int64
	unit     rune
}

func parseExtDuration(value string) (extDuration, error) {
	e := extDuration{}
	if n, err := fmt.Sscanf(value, "%d%c", &e.quantity, &e.unit); n != 2 {
		return e, fmt.Errorf("Invalid value %q for threshold", value)
	} else if err != nil {
		return e, fmt.Errorf("Invalid value %q for threshold: %v", value, err)
	}
	switch e.unit {
	case 'd', 'D', 'w', 'W', 'm', 'M', 'y', 'Y':
		break
	default:
		return e, fmt.Errorf("Invalid unit \"%c\" for threshold", e.unit)
	}
	return e, nil
}

func (e extDuration) Duration() time.Duration {
	day := 24 * time.Hour
	switch e.unit {
	case 'd', 'D':
		return time.Duration(e.quantity) * day
	case 'w', 'W':
		return time.Duration(e.quantity) * 7 * day
	case 'm', 'M':
		return time.Duration(e.quantity) * 31 * day
	case 'y', 'Y':
		return time.Duration(e.quantity) * 356 * day
	default:
		log.Fatalf("Invalid duration unit %c", e.unit)
		return time.Duration(0) // Unreachable
	}
}

func (e extDuration) String() string {
	switch e.unit {
	case 'd', 'D':
		return fmt.Sprintf("%d %s", e.quantity, pluralize(e.quantity, "day"))
	case 'w', 'W':
		return fmt.Sprintf("%d %s", e.quantity, pluralize(e.quantity, "week"))
	case 'm', 'M':
		return fmt.Sprintf("%d %s", e.quantity, pluralize(e.quantity, "month"))
	case 'y', 'Y':
		return fmt.Sprintf("%d %s", e.quantity, pluralize(e.quantity, "year"))
	default:
		log.Fatalf("Invalid duration unit %c", e.unit)
		return "" // Unreachable
	}
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

func parseFilters(filters []string) ([]utils.IssueFilter, error) {
	issueFilters := []utils.IssueFilter{}
	for _, filter := range filters {
		f, err := utils.MakeIssueFilter(filter)
		if err != nil {
			return []utils.IssueFilter{}, err
		}
		issueFilters = append(issueFilters, f)
	}
	return issueFilters, nil
}

func pluralize(count int64, value string) string {
	if count == 1 {
		return value
	}
	return value + "s"
}
