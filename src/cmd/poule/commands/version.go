package commands

import (
	"fmt"
	"regexp"
	"strings"

	"cmd/poule/operations"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

var VersionCommand = cli.Command{
	Name:   "version_label",
	Usage:  "Label issues with the version it applies to",
	Action: doVersionCommand,
}

func doVersionCommand(c *cli.Context) {
	operations.RunIssueOperation(c, &versionOperation{})
}

type versionOperation struct{}

func (o *versionOperation) Apply(c *operations.Context, issue *github.Issue, userData interface{}) error {
	_, _, err := c.Client.Issues.AddLabelsToIssue(c.Username, c.Repository, *issue.Number, []string{userData.(string)})
	return err
}

func (o *versionOperation) Describe(c *operations.Context, issue *github.Issue, userData interface{}) string {
	return fmt.Sprintf("Adding label %q to issue #%d", userData.(string), *issue.Number)
}

func (o *versionOperation) Filter(c *operations.Context, issue *github.Issue) (bool, interface{}) {
	return extractVersionLabels(issue)
}

func (o *versionOperation) ListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func extractVersionLabels(issue *github.Issue) (bool, string) {
	serverVersion := regexp.MustCompile(`Server:\s+Version:\s+(\d+\.\d+\.\d+)-?(\S*)`)
	versionSubmatch := serverVersion.FindStringSubmatch(*issue.Body)
	if len(versionSubmatch) < 3 {
		return false, ""
	}

	label := labelFromVersion(versionSubmatch[1], versionSubmatch[2])
	return true, label
}

func labelFromVersion(version, suffix string) string {
	switch {
	// Dev suffix is associated with a master build.
	case suffix == "dev":
		return "version/master"
	// For a version `X.Y.Z`, add a label of the form `version/X.Y`.
	case strings.HasPrefix(suffix, "cs"):
		fallthrough
	case strings.HasPrefix(suffix, "rc"):
		fallthrough
	case suffix == "":
		return "version/" + version[0:strings.LastIndex(version, ".")]
	// The default for unknown suffix is to consider the version unsupported.
	default:
		return "version/unsupported"
	}
}
