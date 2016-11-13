package gh

import (
	"io/ioutil"

	"poule/configuration"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// DefaultClient is the default implementation for a GitHub client.
type DefaultClient struct {
	Client *github.Client
}

// Issues returns the issue service instance.
func (d DefaultClient) Issues() IssuesService {
	return d.Client.Issues
}

// PullRequests returns the pull request service instance.
func (d DefaultClient) PullRequests() PullRequestsService {
	return d.Client.PullRequests
}

// Repositories returns the repository service instance.
func (d DefaultClient) Repositories() RepositoriesService {
	return d.Client.Repositories
}

// GetToken returns the GitHub API token to use.
func GetToken(c *configuration.Config) string {
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

// MakeClient returns a new client instance for the specified configuration.
func MakeClient(c *configuration.Config) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: GetToken(c)})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return DefaultClient{github.NewClient(tc)}
}
