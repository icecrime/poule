package utils

import (
	"io/ioutil"
	"time"

	"github.com/codegangsta/cli"
	"github.com/google/go-github/github"
)

const (
	BaseUrl        = "https://leeroy.dockerproject.org/build/retry"
	FailingCILabel = "status/failing-ci"
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

type RepoStatus struct {
	CreatedAt time.Time
	State     string
}

func GetLatestStatuses(statuses []github.RepoStatus) map[string]RepoStatus {
	latestStatuses := map[string]RepoStatus{}
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
