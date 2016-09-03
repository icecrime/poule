package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/oauth2"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

const (
	BaseUrl        = "https://leeroy.dockerproject.org/build/retry"
	FailingCILabel = "status/failing-ci"
	PouleToken     = "AUTOMATED:POULE"
)

func GetGitHubToken(c *cli.Context) string {
	if v := c.GlobalString("token"); v != "" {
		return c.GlobalString(v)
	}

	if v := c.GlobalString("token-file"); v != "" {
		if b, err := ioutil.ReadFile(v); err == nil {
			return string(b)
		}
	}

	return ""
}

func HasFailingCILabel(labels []github.Label) bool {
	for _, l := range labels {
		if *l.Name == FailingCILabel {
			return true
		}
	}
	return false
}

func HasFailures(statuses map[string]RepoStatus) bool {
	for _, s := range statuses {
		if s.State != "success" && s.State != "pending" {
			return true
		}
	}
	return false
}

func IsDryRun(c *cli.Context) bool {
	return c.GlobalBool("dry-run")
}

func PrintIssue(issue *github.Issue) {
	labels := []string{}
	for _, label := range issue.Labels {
		labels = append(labels, *label.Name)
	}
	fmt.Printf("Issue #%d\n  Title:  %s\n  Labels: %s\n\n", *issue.Number, *issue.Title, strings.Join(labels, ", "))
}

func GetRepository(c *cli.Context) (string, string) {
	repository := c.GlobalString("repository")
	s := strings.SplitN(repository, "/", 2)
	if len(s) != 2 {
		log.Fatalf("Invalid repository specification %q", repository)
	}
	return s[0], s[1]
}

func MakeGitHubClient(c *cli.Context) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: GetGitHubToken(c)})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}
