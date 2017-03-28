package configuration

// GitHubEvents is the collection of valid GitHub events which can be used as triggers.
var GitHubEvents = []string{
	"commit_comment",
	"create",
	"delete",
	"deployment",
	"deployment_status",
	"fork",
	"gollum",
	"integration_installation",
	"integration_installation_repositories",
	"issue_comment",
	"issues",
	"member",
	"membership",
	"page_build",
	"public",
	"pull_request_review_comment",
	"pull_request",
	"push",
	"repository",
	"release",
	"status",
	"team_add",
	"watch",
}

// StringSlice is a slice of strings.
type StringSlice []string

// Contains returns whether the StringSlice contains a given item.
func (s StringSlice) Contains(item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}
