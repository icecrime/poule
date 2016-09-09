package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"poule/gh"
	"poule/operations"
	"poule/utils"

	"github.com/google/go-github/github"
	"github.com/mitchellh/mapstructure"
	"github.com/urfave/cli"
)

func init() {
	registerOperation(&prRebuildDescriptor{})
}

type prRebuildDescriptor struct{}

func (d *prRebuildDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "rebuild",
		Description: "Rebuild failed pull requests",
	}
}

func (d *prRebuildDescriptor) CommandFlags() []cli.Flag {
	return []cli.Flag{}
}

func (d *prRebuildDescriptor) OperationFromCli(c *cli.Context) operations.Operation {
	return &prRebuildOperation{
		Builder:        rebuildPR,
		Configurations: c.Args(),
	}
}

func (d *prRebuildDescriptor) OperationFromConfig(c operations.Configuration) operations.Operation {
	operation := &prRebuildOperation{
		Builder: rebuildPR,
	}
	if err := mapstructure.Decode(c, &operation); err != nil {
		log.Fatalf("Error creating operation from configuration: %v", err)
	}
	return operation
}

type prRebuildOperation struct {
	Builder        func(pr *github.PullRequest, context string) error
	Configurations []string `mapstructure:"configurations"`
}

func (o *prRebuildOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *prRebuildOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	pr := item.PullRequest()
	for _, context := range userData.([]string) {
		if err := o.Builder(pr, context); err != nil {
			return fmt.Errorf("error rebuilding pull request %d: %v", *pr.Number, err)
		}
	}
	return nil
}

func (o *prRebuildOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	pr := item.PullRequest()
	contexts := userData.([]string)
	if len(contexts) == 0 {
		return ""
	}
	return fmt.Sprintf("Rebuilding pull request #%d for %s", *pr.Number, strings.Join(contexts, ", "))
}

func (o *prRebuildOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	pr := item.PullRequest()
	issue, _, err := c.Client.Issues().Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// Skip all pull requests which are known to fail CI.
	if utils.HasFailingCILabel(issue.Labels) {
		return operations.Reject, nil
	}

	// Get all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories().ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Gather all contexts that need rebuilding.
	contexts := []string{}
	for _, context := range o.Configurations {
		if state := latestStatuses[context].State; state == "error" || state == "failure" {
			contexts = append(contexts, context)
		}
	}
	return operations.Accept, contexts
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

	req, err := http.NewRequest("POST", utils.BaseUrl, bytes.NewBuffer(data))
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
		return fmt.Errorf("requesting %s for PR %d for %s returned status code: %d: make sure the repo allows builds.", utils.BaseUrl, *pr.Number, *pr.Base.Repo.FullName, resp.StatusCode)
	}
	return nil
}
