package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"poule/operations"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

func init() {
	registerOperation(&versionLabelDescriptor{})
}

type versionLabelDescriptor struct{}

func (d *versionLabelDescriptor) Name() string {
	return "version-label"
}

func (d *versionLabelDescriptor) Command() cli.Command {
	return cli.Command{
		Name:  d.Name(),
		Usage: "apply version labels to issues",
		Action: func(c *cli.Context) {
			operations.RunIssueOperation(c, &versionLabel{})
		},
	}
}

func (d *versionLabelDescriptor) Operation() Operation {
	return &versionLabel{}
}

type versionLabel struct{}

func (o *versionLabel) Apply(c *operations.Context, issue *github.Issue, userData interface{}) error {
	_, _, err := c.Client.Issues.AddLabelsToIssue(c.Username, c.Repository, *issue.Number, []string{userData.(string)})
	return err
}

func (o *versionLabel) Describe(c *operations.Context, issue *github.Issue, userData interface{}) string {
	return fmt.Sprintf("Adding label %q to issue #%d", userData.(string), *issue.Number)
}

func (o *versionLabel) Filter(c *operations.Context, issue *github.Issue) (bool, interface{}) {
	return extractVersionLabels(issue)
}

func (o *versionLabel) ListOptions(c *operations.Context) *github.IssueListByRepoOptions {
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
