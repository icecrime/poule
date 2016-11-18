package gh

import "github.com/google/go-github/github"

// Client allows us to wrap the use of the go-github library in order to
// be able to mock it in tests.
type Client interface {
	Issues() IssuesService
	PullRequests() PullRequestsService
	Repositories() RepositoriesService
	Search() SearchService
}

// IssuesService is the interface to the GitHub issue service.
//go:generate mockery -name=IssuesService -output ../test/mocks
type IssuesService interface {
	// Issue API.
	Edit(owner string, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
	Get(owner string, repo string, number int) (*github.Issue, *github.Response, error)
	ListByRepo(owner string, repo string, opt *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)

	// Comments API.
	CreateComment(owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	DeleteComment(owner string, repo string, id int) (*github.Response, error)
	ListComments(owner string, repo string, number int, opt *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)

	// Label API.
	AddLabelsToIssue(owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	RemoveLabelForIssue(owner string, repo string, number int, label string) (*github.Response, error)

	// Milestones API.
	ListMilestones(owner string, repo string, opt *github.MilestoneListOptions) ([]*github.Milestone, *github.Response, error)
}

// PullRequestsService is the interface to the GitHub pull request service.
//go:generate mockery -name=PullRequestsService -output ../test/mocks
type PullRequestsService interface {
	// Pull requests API.
	Get(owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
	List(owner string, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListFiles(owner string, repo string, number int, opt *github.ListOptions) ([]*github.CommitFile, *github.Response, error)

	// Commits API.
	ListCommits(owner string, repo string, number int, opt *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
}

// RepositoriesService is the interface to the GitHub repositories service.
//go:generate mockery -name=RepositoriesService -output ../test/mocks
type RepositoriesService interface {
	// Statuses API.
	CreateStatus(owner, repo, ref string, sts *github.RepoStatus) (*github.RepoStatus, *github.Response, error)
	ListStatuses(owner, repo, ref string, opt *github.ListOptions) ([]*github.RepoStatus, *github.Response, error)
}

// SearchService is the interface to the GitHub search service.
//go:generate mockery -name=SearchService -output ../test/mocks
type SearchService interface {
	Issues(query string, opt *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error)
}
