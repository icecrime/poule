package utils

import (
	"log"
	"time"

	"github.com/google/go-github/github"
)

type RepoStatus struct {
	CreatedAt time.Time
	State     string
}

type StatusSnapshot map[string]RepoStatus

func (s *StatusSnapshot) Print(pr *github.PullRequest) {
	log.Printf("Statuses for PR#%d (%s)\n", *pr.Number, *pr.Head.SHA)
	for context, repoStatus := range *s {
		log.Printf("  %-30s%s\n", context, repoStatus.State)
	}
}

func GetLatestStatuses(statuses []github.RepoStatus) StatusSnapshot {
	latestStatuses := StatusSnapshot{}
	for _, repoStatus := range statuses {
		if repoStatus.CreatedAt.Unix() > latestStatuses[*repoStatus.Context].CreatedAt.Unix() {
			latestStatuses[*repoStatus.Context] = RepoStatus{
				CreatedAt: *repoStatus.CreatedAt,
				State:     *repoStatus.State,
			}
		}
	}
	return latestStatuses
}
