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

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

func init() {
	registerOperation(&prRebuildDescriptor{})
}

type prRebuildDescriptor struct{}

func (d *prRebuildDescriptor) Name() string {
	return "rebuild"
}

func (d *prRebuildDescriptor) Command() cli.Command {
	return cli.Command{
		Name:  d.Name(),
		Usage: "rebuild failed pull requests",
		Action: func(c *cli.Context) {
			operations.RunPullRequestOperation(c, &prRebuild{
				args: c.Args(),
			})
		},
	}
}

func (d *prRebuildDescriptor) Operation() Operation {
	return &ciFailureLabelAudit{}
}

type prRebuild struct {
	args cli.Args
}

func (o *prRebuild) Apply(c *operations.Context, pr *github.PullRequest, userData interface{}) error {
	for _, context := range userData.([]string) {
		if err := rebuildPR(pr, context); err != nil {
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

func (o *prRebuild) Filter(c *operations.Context, pr *github.PullRequest) (bool, interface{}) {
	// Fetch the issue information for that pull request: that's the only way
	// to retrieve the labels.
	issue, _, err := c.Client.Issues.Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatalf("Error getting issue %d: %v", *pr.Number, err)
	}

	// Skip all pull requests which are known to fail CI.
	if utils.HasFailingCILabel(issue.Labels) {
		return false, nil
	}

	// Get all statuses for that item.
	repoStatuses, _, err := c.Client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}
	latestStatuses := utils.GetLatestStatuses(repoStatuses)

	// Gather all contexts that need rebuilding.
	contexts := []string{}
	for _, context := range o.args {
		if state := latestStatuses[context].State; state == "error" || state == "failure" {
			contexts = append(contexts, context)
		}
	}
	return true, contexts
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
