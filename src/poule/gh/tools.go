package gh

import (
	"poule/configuration"

	"github.com/google/go-github/github"
)

func HasFailingCILabel(labels []github.Label) bool {
	for _, l := range labels {
		if *l.Name == configuration.FailingCILabel {
			return true
		}
	}
	return false
}
