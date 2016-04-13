package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"cmd/poule/utils"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var RebuildCommand = cli.Command{
	Name:   "rebuild",
	Usage:  "rebuild failed jobs",
	Action: doRebuildCommand,
}

type PullRequest struct {
	Number  int    `json:"number"`
	Repo    string `json:"repo"`
	Context string `json:"context"`
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
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("Requesting %s for PR %d for %s returned status code: %d: make sure the repo allows builds.", utils.BaseUrl, *pr.Number, *pr.Base.Repo.FullName, resp.StatusCode)
	}
	return nil
}

func handlePR(contexts []string, client *github.Client, pr *github.PullRequest) {
	issue, _, err := client.Issues.Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
	if err != nil {
		log.Fatal(err)
	}
	for _, l := range issue.Labels {
		if *l.Name == utils.FailingCILabel {
			log.Printf("Skipping PR#%d (known to fail CI)", *pr.Number)
			return
		}
	}

	repoStatuses, _, err := client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
	if err != nil {
		log.Fatal(err)
	}

	latestStatuses := utils.GetLatestStatuses(repoStatuses)
	log.Printf("Statuses for PR#%d (%s)\n", *pr.Number, *pr.Head.SHA)
	for context, repoStatus := range latestStatuses {
		log.Printf("  %-30s%s\n", context, repoStatus.State)
	}

	for _, context := range contexts {
		if state := latestStatuses[context].State; state == "error" || state == "failure" {
			log.Printf("* Rebuilding PR#%d for %q\n", *pr.Number, context)
			if err := rebuildPR(pr, context); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func doRebuildCommand(c *cli.Context) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: utils.GetGitHubToken(c)})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	repo := strings.SplitN(c.GlobalString("repository"), "/", 2)
	opts := github.PullRequestListOptions{
		State: "open",
		Base:  "master",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 200,
		},
	}

	prs, _, err := client.PullRequests.List(repo[0], repo[1], &opts)
	if err != nil {
		log.Fatal(err)
	}

	contexts := c.Args()
	for _, pr := range prs {
		handlePR(contexts, client, &pr)
	}
}
