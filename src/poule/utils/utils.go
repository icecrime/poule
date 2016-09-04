package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"poule/configuration"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	BaseUrl        = "https://leeroy.dockerproject.org/build/retry"
	FailingCILabel = "status/failing-ci"
	PouleToken     = "AUTOMATED:POULE"
)

func GetGitHubToken(c *configuration.Config) string {
	if c.Token != "" {
		return c.Token
	}

	if c.TokenFile != "" {
		if b, err := ioutil.ReadFile(c.TokenFile); err == nil {
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

func IsDryRun(c *configuration.Config) bool {
	return c.DryRun
}

func PrintIssue(issue *github.Issue) {
	labels := []string{}
	for _, label := range issue.Labels {
		labels = append(labels, *label.Name)
	}
	fmt.Printf("Issue #%d\n  Title:  %s\n  Labels: %s\n\n", *issue.Number, *issue.Title, strings.Join(labels, ", "))
}

func GetRepository(c *configuration.Config) (string, string) {
	s := strings.SplitN(c.Repository, "/", 2)
	if len(s) != 2 {
		log.Fatalf("Invalid repository specification %q", c.Repository)
	}
	return s[0], s[1]
}

func MakeGitHubClient(c *configuration.Config) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: GetGitHubToken(c)})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}
