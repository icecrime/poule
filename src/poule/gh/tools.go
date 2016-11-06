package gh

import (
	"poule/configuration"

	"github.com/google/go-github/github"
)

func HasLabel(label string, issueLabels []github.Label) bool {
	for _, issueLabel := range issueLabels {
		if label == *issueLabel.Name {
			return true
		}
	}
	return false
}

func HasAnyLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if HasLabel(label, issueLabels) {
			return true
		}
	}
	return false
}

func HasAllLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if !HasLabel(label, issueLabels) {
			return false
		}
	}
	return true
}

func HasFailingCILabel(labels []github.Label) bool {
	for _, l := range labels {
		if *l.Name == configuration.FailingCILabel {
			return true
		}
	}
	return false
}
