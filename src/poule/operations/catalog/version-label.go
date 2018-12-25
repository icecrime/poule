package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&versionLabelDescriptor{})
}

type versionLabelDescriptor struct{}

func (d *versionLabelDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "version-label",
		Description: "Apply version labels to issues",
	}
}

func (d *versionLabelDescriptor) OperationFromCli(*cli.Context) (operations.Operation, error) {
	return &versionLabelOperation{}, nil
}

func (d *versionLabelDescriptor) OperationFromConfig(operations.Configuration) (operations.Operation, error) {
	return &versionLabelOperation{}, nil
}

type versionLabelOperation struct{}

func (o *versionLabelOperation) Accepts() operations.AcceptedType {
	return operations.Issues
}

func (o *versionLabelOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	issue := item.Issue
	_, _, err := c.Client.Issues().AddLabelsToIssue(c.Username, c.Repository, *issue.Number, []string{userData.(string)})
	return err
}

func (o *versionLabelOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("adding label %q", userData.(string))
}

func (o *versionLabelOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	issue := item.Issue
	if b, label := extractVersionLabels(issue); b {
		return operations.Accept, label, nil
	}
	return operations.Reject, nil, nil
}

func (o *versionLabelOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func (o *versionLabelOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	// versionLabelOperation doesn't apply to GitHub pull requests.
	return nil
}

func extractVersionLabels(issue *github.Issue) (bool, string) {
	if issue.Body == nil {
		return false, ""
	}
	serverVersion := regexp.MustCompile(`(?:Server:?)\s+(?:Docker Engine - Community\s+)?(?:Engine:\s+)?(?:Version:\s+)(\d+\.\d+\.\d+)(-\w+)?`)
	versionSubmatch := serverVersion.FindStringSubmatch(*issue.Body)
	if len(versionSubmatch) < 3 {
		return false, ""
	}
	label := labelFromVersion(versionSubmatch[1], strings.TrimPrefix(versionSubmatch[2], "-"))
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
	case strings.HasPrefix(suffix, "ce"):
		fallthrough
	case strings.HasPrefix(suffix, "ee"):
		fallthrough
	case suffix == "":
		return "version/" + version[0:strings.LastIndex(version, ".")]
	// The default for unknown suffix is to consider the version unsupported.
	default:
		return "version/unsupported"
	}
}
