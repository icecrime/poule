package commands

import (
	"log"
	"strings"

	"cmd/poule/utils"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var CleanCommand = cli.Command{
	Name:   "clean",
	Usage:  "clean github failure labels",
	Action: doCleanCommand,
}

func doCleanCommand(c *cli.Context) {
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

	for _, pr := range prs {
		issue, _, err := client.Issues.Get(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number)
		if err != nil {
			log.Fatal(err)
		}

		var hasFailingCILabel bool
		for _, l := range issue.Labels {
			if *l.Name == utils.FailingCILabel {
				hasFailingCILabel = true
				break
			}
		}

		if !hasFailingCILabel {
			return
		}

		repoStatuses, _, err := client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
		if err != nil {
			log.Fatal(err)
		}
		latestStatuses := utils.GetLatestStatuses(repoStatuses)

		var hasFailures bool
		for _, s := range latestStatuses {
			if s.State != "success" {
				hasFailures = true
				break
			}
		}
		if hasFailures {
			log.Printf("PR#%d still has failures: keeping label %q", *pr.Number, utils.FailingCILabel)
		} else {
			log.Printf("Removing label %q from PR#%d", utils.FailingCILabel, *pr.Number)
			if _, err := client.Issues.RemoveLabelForIssue(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, utils.FailingCILabel); err != nil {
				log.Fatal(err)
			}
		}
	}
}
