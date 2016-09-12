package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"poule/configuration"
	"poule/gh"
	"poule/operations"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&prRebuildDescriptor{})
}

type prRebuildDescriptor struct{}

func (d *prRebuildDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "rebuild",
		Description: "Rebuild the specified configurations when in failure",
		ArgsUsage:   "configuration [configuration...]",
	}
}

func (d *prRebuildDescriptor) CommandFlags() []cli.Flag {
	return []cli.Flag{}
}

func (d *prRebuildDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	if c.NArg() < 1 {
		return nil, errors.Errorf("rebuild requires at least one argument")
	}
	return &prRebuildOperation{
		Builder:        rebuildPR,
		Configurations: c.Args(),
	}, nil
}

func (d *prRebuildDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	operation := &prRebuildOperation{
		Builder: rebuildPR,
	}
	if err := mapstructure.Decode(c, &operation); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	return operation, nil
}

type prRebuildOperation struct {
	Builder        func(pr *github.PullRequest, context string) error
	Configurations []string `mapstructure:"configurations"`
}

func (o *prRebuildOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *prRebuildOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	pr := item.PullRequest
	for _, context := range userData.([]string) {
		if err := o.Builder(pr, context); err != nil {
			return fmt.Errorf("error rebuilding pull request %d: %v", *pr.Number, err)
		}
	}
	return nil
}

func (o *prRebuildOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	contexts := userData.([]string)
	if len(contexts) == 0 {
		return ""
	}
	return fmt.Sprintf("Rebuilding pull request #%d for %s", item.Number(), strings.Join(contexts, ", "))
}

func (o *prRebuildOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	pr := item.PullRequest
	if _, err := item.GetRelatedIssue(c.Client); err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve issue #%d", *pr.Number)
	}

	// Skip all pull requests which are known to fail CI.
	if gh.HasFailingCILabel(item.Issue.Labels) {
		return operations.Reject, nil, nil
	}

	// Get all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve statuses for pull request #%d", *pr.Number)
	}
	latestStatuses := gh.GetLatestStatuses(repoStatuses)

	// Gather all contexts that need rebuilding.
	contexts := []string{}
	for _, context := range o.Configurations {
		if state := latestStatuses[context].State; state == "error" || state == "failure" {
			contexts = append(contexts, context)
		}
	}
	return operations.Accept, contexts, nil
}

func (o *prRebuildOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	// prRebuildOperation doesn't apply to GitHub issues.
	return nil
}

func (o *prRebuildOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return &github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			PerPage: 200,
		},
	}
}

func rebuildPR(pr *github.PullRequest, context string) (err error) {
	prData := map[string]interface{}{
		"number":  *pr.Number,
		"repo":    pr.Base.Repo.FullName,
		"context": context,
	}
	data, err := json.Marshal(prData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", configuration.JenkinsBaseUrl, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(os.Getenv("LEEROY_USERNAME"), os.Getenv("LEEROY_PASS"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return errors.Errorf("requesting %s for PR %d for %s returned status code: %d: make sure the repo allows builds.", configuration.JenkinsBaseUrl, *pr.Number, *pr.Base.Repo.FullName, resp.StatusCode)
	}
	return nil
}
