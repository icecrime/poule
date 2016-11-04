package gh

import (
	"time"

	"github.com/google/go-github/github"
)

// RepoStatus is the status of a repository at a particular reference.
type RepoStatus struct {
	CreatedAt time.Time
	State     string
}

// StatusSnapshot is a collection of statuses indexed by the corresponding
// configuration name.
type StatusSnapshot map[string]RepoStatus

// HasFailures returns any whether any of the statuses are in a failed state.
func (s *StatusSnapshot) HasFailures() bool {
	for _, r := range *s {
		if r.State != "success" && r.State != "pending" {
			return true
		}
	}
	return false
}

// GetLatestStatuses returns the selection of the latest status for each
// configuration.
func GetLatestStatuses(statuses []*github.RepoStatus) StatusSnapshot {
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
