package catalog

import (
	"fmt"
	"log"
	"poule/operations"
	"strconv"
	"strings"
	"time"

	"poule/utils"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

func init() {
	registerOperation(&pruneDescriptor{})
}

func doRunPrune(c *cli.Context) {
	action := c.String("action")
	switch action {
	case "close", "ping", "warn":
		break
	default:
		log.Fatalf("Invalid action %q", action)
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

	operations.RunIssueOperation(c, &prune{
		action:            action,
		filters:           filters,
		gracePeriod:       gracePeriod,
		outdatedThreshold: outdatedThreshold,
	})
}

type pruneDescriptor struct{}

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
				Usage: "action to take for outdated issues (\"ping\", \"warn\", or \"close\")",
				Value: "ping",
			},
			cli.StringSliceFlag{
				Name:  "filter",
				Usage: "filter issue attributes (\"label=value\", or \"assigned=bool\"))",
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

func (d *pruneDescriptor) Operation() Operation {
	return &prune{}
}

type prune struct {
	action            string
	filters           issueFilters
	gracePeriod       extDuration
	outdatedThreshold extDuration
}

func (o *prune) Apply(c *operations.Context, issue *github.Issue, userData interface{}) error {
	return nil
}

func (o *prune) Describe(c *operations.Context, issue *github.Issue, userData interface{}) string {
	labels := []string{}
	for _, label := range issue.Labels {
		labels = append(labels, *label.Name)
	}
	s := fmt.Sprintf("Issue #%d is outdated\n  Last:   %v\n  Title:  %s\n  Labels: %s\n\n", *issue.Number, userData.(time.Time), *issue.Title, strings.Join(labels, ", "))
	switch o.action {
	case "close":
		break
	case "ping":
		//fmt.Printf("\n%s\n\n", formatPingComment(issue, o))
		break
	case "warn":
		//fmt.Printf("\n%s\n\n", formatWarnComment(issue, o))
		break
	}
	return s
}

func (o *prune) Filter(c *operations.Context, issue *github.Issue) (bool, interface{}) {
	// Apply filters, if any.
	if o.filters.assigned != nil && (*o.filters.assigned != (issue.Assignee != nil)) {
		return false, nil
	}
	if !hasAllLabels(o.filters.labels, issue.Labels) || hasAnyLabels(o.filters.not_labels, issue.Labels) {
		return false, nil
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

	// Filter out issues which last commented date is under our threshold.
	outdated := lastCommented.Add(o.outdatedThreshold.Duration()).Before(time.Now())
	return outdated, lastCommented
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
	comment := `<!-- %s -->
@%s Hi! We see that this issue has not received any activity in over %s.

Could you please let us know if it is still relevant? For example:
- For a bug: do you still experience the issue with the latest version?
- For a feature request: was your request appropriately answered in a later version?
`
	return fmt.Sprintf(comment, utils.PouleToken, *issue.User.Login, o.outdatedThreshold.String())
}

func formatWarnComment(issue *github.Issue, o *prune) string {
	comment := `%s
Thank you very much for your help! The issue will be **automatically closed in %s** unless it is commented on.
`
	base := formatPingComment(issue, o)
	return fmt.Sprintf(comment, base, o.gracePeriod.String())
}

func hasLabel(s string, issueLabels []github.Label) bool {
	for _, label := range issueLabels {
		if s == *label.Name {
			return true
		}
	}
	return false
}

func hasAnyLabels(s []string, issueLabels []github.Label) bool {
	for _, l := range s {
		if hasLabel(l, issueLabels) {
			return true
		}
	}
	return false
}

func hasAllLabels(s []string, issueLabels []github.Label) bool {
	for _, l := range s {
		if !hasLabel(l, issueLabels) {
			return false
		}
	}
	return true
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

type issueFilters struct {
	assigned   *bool
	labels     []string
	not_labels []string
}

func parseFilters(filters []string) (issueFilters, error) {
	result := issueFilters{}
	for _, filter := range filters {
		s := strings.SplitN(filter, "=", 2)
		if len(s) != 2 {
			return issueFilters{}, fmt.Errorf("Invalid filter %q", s)
		}

		switch s[0] {
		case "assigned":
			b, err := strconv.ParseBool(s[1])
			if err != nil {
				return issueFilters{}, fmt.Errorf("Invalid value %q for assigned", s[1])
			}
			result.assigned = &b
		case "label":
			result.labels = append(result.labels, s[1])
		case "~label":
			result.not_labels = append(result.not_labels, s[1])
		default:
			return issueFilters{}, fmt.Errorf("Unknown filter type %q", s[0])
		}
	}
	return result, nil
}

func pluralize(count int64, value string) string {
	if count == 1 {
		return value
	}
	return value + "s"
}