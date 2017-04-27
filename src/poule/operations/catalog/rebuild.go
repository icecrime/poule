package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"poule/common"
	"poule/configuration"
	"poule/gh"
	"poule/operations"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var defaultStatuses = []string{"error", "failure"}

func init() {
	registerOperation(&prRebuildDescriptor{})
}

type prRebuildDescriptor struct{}

func (d *prRebuildDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "rebuild",
		Description: "Rebuild configurations of a given state",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "configurations",
				Usage: "configurations to rebuild (empty means all)",
			},
			cli.StringFlag{
				Name:  "label",
				Usage: "label to search for and remove after rebuild",
			},
			cli.StringSliceFlag{
				Name:  "status",
				Usage: "status filter of configurations to rebuild",
				Value: (*cli.StringSlice)(&defaultStatuses),
			},
		},
	}
}

func (d *prRebuildDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	return &prRebuildOperation{
		Builder:        rebuildPR,
		Configurations: c.StringSlice("configurations"),
		Label:          c.String("label"),
		Statuses:       c.StringSlice("status"),
	}, nil
}

func (d *prRebuildDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	operation := &prRebuildOperation{}
	if err := mapstructure.Decode(c, &operation); err != nil {
		return nil, errors.Wrap(err, "decoding configuration")
	}
	// Set the default value for statuses.
	operation.Builder = rebuildPR
	if len(operation.Statuses) == 0 {
		operation.Statuses = defaultStatuses
	}
	return operation, nil
}

type prRebuildOperation struct {
	Builder        func(pr *github.PullRequest, context string) error
	Label          string   `mapstructure:"label"`
	Statuses       []string `mapstructure:"statuses"`
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

	// Remove our trigger label, if any.
	var err error
	if o.Label != "" {
		_, err = c.Client.Issues().RemoveLabelForIssue(c.Username, c.Repository, *pr.Number, o.Label)
	}
	return err
}

func (o *prRebuildOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	contexts := userData.([]string)
	if len(contexts) == 0 {
		return ""
	}
	return fmt.Sprintf("Rebuilding pull request #%d for %q", item.Number(), strings.Join(contexts, ", "))
}

func (o *prRebuildOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Never rebuild a close pull request.
	pr := item.PullRequest
	if pr.State != nil && *pr.State != "open" {
		logrus.Debugf("rejecting pull request with state=%q", *pr.State)
		return operations.Reject, nil, nil
	}

	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	if _, err := item.GetRelatedIssue(c.Client); err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve issue #%d", *pr.Number)
	}

	// Search for our trigger label, if specified.
	if o.Label != "" && !gh.HasLabel(o.Label, item.Issue.Labels) {
		logrus.Debugf("rejecting pull request without label %q", o.Label)
		return operations.Reject, nil, nil
	}

	// Get all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		return operations.Reject, nil, errors.Wrapf(err, "failed to retrieve statuses for pull request #%d", *pr.Number)
	}

	// Gather all contexts which need rebuilding. When no configuration is specified, we used the
	// list of statuses currently associated with the pull request as the set to be rebuilt. When
	// specific configurations are specified, we allow rebuilding those even they had never been
	// run yet.
	//
	// The set of possible configurations to rebuild is either the specified one or the set of
	// currently executed configurations for that pull request.
	rebuildCandidates := o.Configurations
	latestStatuses := gh.GetLatestStatuses(repoStatuses)
	if len(rebuildCandidates) == 0 {
		for config := range latestStatuses {
			rebuildCandidates = append(rebuildCandidates, config)
		}
	}

	// For each possible configuration to rebuild, we verify if the status is eligible (if any).
	contexts := []string{}
	for _, config := range rebuildCandidates {
		if status, ok := latestStatuses[config]; !ok || common.ContainsString(o.Statuses, status.State) {
			contexts = append(contexts, config)
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

	req, err := http.NewRequest("POST", configuration.JenkinsBaseURL, bytes.NewBuffer(data))
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
		return errors.Errorf("requesting %s for PR %d for %s returned status code: %d: make sure the repo allows builds.", configuration.JenkinsBaseURL, *pr.Number, *pr.Base.Repo.FullName, resp.StatusCode)
	}
	return nil
}
