package gh

import (
	"io/ioutil"

	"poule/configuration"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type DefaultClient struct {
	Client *github.Client
}

func (d DefaultClient) Issues() IssuesService {
	return d.Client.Issues
}

func (d DefaultClient) PullRequests() PullRequestsService {
	return d.Client.PullRequests
}

func (d DefaultClient) Repositories() RepositoriesService {
	return d.Client.Repositories
}

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

func MakeClient(c *configuration.Config) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: GetToken(c)})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return DefaultClient{github.NewClient(tc)}
}
