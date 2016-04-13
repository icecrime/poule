package commands

import (
	"log"
	"strings"

	"cmd/poule/utils"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var AuditCommand = cli.Command{
	Name:   "audit",
	Usage:  "audit github jobs failure",
	Action: doAuditCommand,
}

func doAuditCommand(c *cli.Context) {
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
		p, _, err := client.PullRequests.Get(repo[0], repo[1], *pr.Number)
		if err != nil {
			log.Fatal(err)
		}
		if p.Mergeable != nil && !*p.Mergeable {
			continue
		}

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

		repoStatuses, _, err := client.Repositories.ListStatuses(*pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Head.SHA, nil)
		if err != nil {
			log.Fatal(err)
		}
		latestStatuses := utils.GetLatestStatuses(repoStatuses)

		var hasFailures bool
		for _, s := range latestStatuses {
			if s.State != "success" && s.State != "pending" {
				hasFailures = true
				break
			}
		}

		if hasFailingCILabel && !hasFailures {
			log.Printf("PR#%d is labeled %q but has no failures", *pr.Number, utils.FailingCILabel)
		} else if !hasFailingCILabel && hasFailures {
			log.Printf("PR#%d is not labeled %q but has failures", *pr.Number, utils.FailingCILabel)
		}
	}
}
