package operations

import "github.com/google/go-github/github"

type Context struct {
	// Client is the GitHub API client instance.
	Client *github.Client

	// Username is the owner of the GitHub repository.
	Username string

	// Repository is the name of the GitHub repository.
	Repository string
}
