package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

func (d *prRebuildDescriptor) Description() string {
	return "Rebuild failed pull requests"
}

func (d *prRebuildDescriptor) Flags() []cli.Flag {
	return nil
}

func (d *prRebuildDescriptor) Name() string {
	return "rebuild"
}

func (d *prRebuildDescriptor) CommandFlags() []cli.Flag {
	return []cli.Flag{}
}

func (d *prRebuildDescriptor) OperationFromCli(c *cli.Context) Operation {
	return &prRebuild{
		Builder:        rebuildPR,
		Configurations: c.Args(),
	}
}

func (d *prRebuildDescriptor) OperationFromConfig(c operations.Configuration) Operation {
	operation := &prRebuild{
		Builder: rebuildPR,
	}
	if err := mapstructure.Decode(c, &operation); err != nil {
		log.Fatalf("Error creating operation from configuration: %v", err)
	}
	return operation
}

type prRebuild struct {
	Builder        func(pr *github.PullRequest, context string) error
	Configurations []string `mapstructure:"configurations"`
}

func (o *prRebuild) Apply(c *operations.Context, pr *github.PullRequest, userData interface{}) error {
	for _, context := range userData.([]string) {
		if err := o.Builder(pr, context); err != nil {
			return fmt.Errorf("error rebuilding pull request %d: %v", *pr.Number, err)
		}
	}
	return nil
}

func (o *prRebuild) Describe(c *operations.Context, pr *github.PullRequest, userData interface{}) string {
	contexts := userData.([]string)
	if len(contexts) == 0 {
		return ""
	}
	return fmt.Sprintf("Rebuilding pull request #%d for %s", *pr.Number, strings.Join(contexts, ", "))
}

func (o *prRebuild) Filter(c *operations.Context, pr *github.PullRequest) (operations.FilterResult, interface{}) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
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

func (o *prRebuild) ListOptions(c *operations.Context) *github.PullRequestListOptions {
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
