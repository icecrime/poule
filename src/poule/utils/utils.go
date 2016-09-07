package utils

import (
	"fmt"
	"strings"

	"poule/configuration"

	"github.com/google/go-github/github"
)

const (
	BaseUrl        = "https://leeroy.dockerproject.org/build/retry"
	FailingCILabel = "status/failing-ci"
	PouleToken     = "AUTOMATED:POULE"
)

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
