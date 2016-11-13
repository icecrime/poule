package gh

import (
	"poule/configuration"

	"github.com/google/go-github/github"
)

// HasLabel returns true if a collection of labels contains the specified label.
func HasLabel(label string, issueLabels []github.Label) bool {
	for _, issueLabel := range issueLabels {
		if label == *issueLabel.Name {
			return true
		}
	}
	return false
}

// HasAnyLabels returns true if a collection of labels contains any of the specified collection of
// labels.
func HasAnyLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if HasLabel(label, issueLabels) {
			return true
		}
	}
	return false
}

// HasAllLabels returns true if a collection of labels contains all of the specified collection of
// labels.
func HasAllLabels(labels []string, issueLabels []github.Label) bool {
	for _, label := range labels {
		if !HasLabel(label, issueLabels) {
			return false
		}
	}
	return true
}

// HasFailingCILabel returns true if a collection of labels contains the particular label indicating
// test failures.
func HasFailingCILabel(labels []github.Label) bool {
	for _, l := range labels {
		if *l.Name == configuration.FailingCILabel {
			return true
		}
	}
	return false
}
