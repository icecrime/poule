package catalog

import (
	"fmt"
	"log"
	"poule/operations"
	"strings"
	"time"

	"poule/utils"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&pruneDescriptor{})
}

type pruneDescriptor struct{}

type pruneConfig struct {
	Action            string                 `mapstructure:"action"`
	Filters           pruneFilterDescription `mapstructure:"filters"`
	GracePeriod       string                 `mapstructure:"grace-period"`
	OutdatedThreshold string                 `mapstructure:"outdated-threshold"`
}

type pruneFilterDescription map[string][]string

func (d *pruneDescriptor) Description() string {
	return "Prune outdated issues"
}

func (d *pruneDescriptor) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "action",
			Usage: "action to take for outdated issues",
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
			Usage: "threshold in days, weeks, months, or years",
			Value: "6m",
		},
	}
}

func (d *pruneDescriptor) Name() string {
	return "prune"
}

func (d *pruneDescriptor) OperationFromCli(c *cli.Context) Operation {
	pruneConfig := &pruneConfig{
		Action:            c.String("action"),
		Filters:           pruneFilterDescription{},
		GracePeriod:       c.String("grace-period"),
		OutdatedThreshold: c.String("threshold"),
	}
	for _, filter := range c.StringSlice("filter") {
		s := strings.SplitN(filter, ":", 2)
		if len(s) != 2 {
			log.Fatalf("Invalid filter format %q", filter)
		}
		pruneConfig.Filters[s[0]] = strings.Split(s[1], ",")
	}
	return d.makeOperation(pruneConfig)
}

func (d *pruneDescriptor) OperationFromConfig(c operations.Configuration) Operation {
	pruneConfig := &pruneConfig{}
	if err := mapstructure.Decode(c, &pruneConfig); err != nil {
		log.Fatalf("Error creating operation from configuration: %v", err)
	}
	return d.makeOperation(pruneConfig)
}

func (d *pruneDescriptor) makeOperation(config *pruneConfig) Operation {
	var (
		err       error
		operation prune
	)
	if operation.action, err = parseAction(config.Action); err != nil {
		log.Fatal(err)
	}
	if operation.filters, err = parseFilters(config.Filters); err != nil {
		log.Fatal(err)
	}
	if operation.gracePeriod, err = utils.ParseExtDuration(config.GracePeriod); err != nil {
		log.Fatal(err)
	}
	if operation.outdatedThreshold, err = utils.ParseExtDuration(config.OutdatedThreshold); err != nil {
		log.Fatal(err)
	}
	return &operation
}

type prune struct {
	action            string
	filters           []utils.IssueFilter
	gracePeriod       utils.ExtDuration
	outdatedThreshold utils.ExtDuration
}

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
		o.outdatedThreshold.Quantity,
		o.outdatedThreshold.Unit,
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

func parseAction(action string) (string, error) {
	switch action {
	case "close", "force-close", "ping", "warn":
		break
	default:
		return "", fmt.Errorf("Invalid action %q", action)
	}
	return action, nil
}

func parseFilters(filters map[string][]string) ([]utils.IssueFilter, error) {
	issueFilters := []utils.IssueFilter{}
	for filterType, value := range filters {
		f, err := utils.MakeIssueFilter(filterType, strings.Join(value, ","))
		if err != nil {
			return []utils.IssueFilter{}, err
		}
		issueFilters = append(issueFilters, f)
	}
	return issueFilters, nil
}
