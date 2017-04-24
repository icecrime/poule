package catalog

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"poule/gh"
	"poule/operations"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

// DockerVersionURL is the URL for the Docker VERSION file.
const DockerVersionURL = "https://raw.githubusercontent.com/moby/moby/master/VERSION"

func init() {
	registerOperation(&versionMilestoneDescriptor{})
}

type versionMilestoneDescriptor struct{}

func (d *versionMilestoneDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "version-milestone",
		Description: "Attach merged pull requests to the upcoming version's milestone",
	}
}

func (d *versionMilestoneDescriptor) OperationFromCli(*cli.Context) (operations.Operation, error) {
	return d.OperationFromConfig(nil)
}

func (d *versionMilestoneDescriptor) OperationFromConfig(operations.Configuration) (operations.Operation, error) {
	return &versionMilestoneOperation{
		VersionGetter: getVersionFromRepository,
	}, nil
}

type versionMilestoneOperation struct {
	VersionGetter func(repository string) (string, error)
}

func (o *versionMilestoneOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *versionMilestoneOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	_, _, err := c.Client.Issues().Edit(c.Username, c.Repository, item.Number(), &github.IssueRequest{
		Milestone: userData.(*github.Milestone).Number,
	})
	return err
}

func (o *versionMilestoneOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	return fmt.Sprintf("adding pull request to milestone %d (%q)", *userData.(*github.Milestone).Number, *userData.(*github.Milestone).Title)
}

func (o *versionMilestoneOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// We only consider merged pull requests against the master branch which don't already have a
	// milestone set.
	pr := item.PullRequest
	switch {
	case pr.Merged != nil && *pr.Merged == false:
		logrus.Debug("rejecting unmerged pull request")
		return operations.Reject, nil, nil
	case pr.Milestone != nil:
		logrus.Debugf("rejecting pull request with milestone %d (%q)", *pr.Milestone.Number, *pr.Milestone.Title)
		return operations.Reject, nil, nil
	case *pr.Base.Ref != "master":
		logrus.Debugf("rejecting pull request against non-master branch %q", *pr.Base.Ref)
		return operations.Reject, nil, nil
	}

	// We need to find the milestone that pull request belongs to: we get the VERSION file at the
	// root of the repository, and try to find a matching milestone from there.
	version, err := o.VersionGetter(item.Repository())
	if err != nil {
		return operations.Reject, nil, err
	}

	// Try to find a milestone with the corresponding name. GitHub API doesn't give us a way to
	// search milestones by name so we need to retrieve all open ones.
	milestones, _, err := c.Client.Issues().ListMilestones(c.Username, c.Repository, nil)
	if err != nil {
		return operations.Reject, nil, err
	}

	// Find the matchin milestone
	var targetMilestone *github.Milestone
	for _, milestone := range milestones {
		if *milestone.Title == version {
			targetMilestone = milestone
			break
		}
	}

	// Accept the pull request if we successfully found the target milestone.
	if targetMilestone == nil {
		logrus.Debugf("failed to find matching milestone for version %q", version)
		return operations.Reject, nil, nil
	}
	return operations.Accept, targetMilestone, nil
}

func (o *versionMilestoneOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	// versionMilestoneOperation doesn't apply to GitHub issues.
	return nil
}

func (o *versionMilestoneOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	// versionMilestoneOperation is a dangerous one to run as batch, as it will take all previously
	// merged pull requests and associate them with the next milestone. Timing matters for this
	// operation, and that would be a mistake to do so. Returning nil here is our way to disable
	// batch invokation for this operation.
	return nil
}

func getVersionFromRepository(repository string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/%s/master/VERSION", repository))
	if err != nil {
		return "", fmt.Errorf("failed to retrieve version from %q: %v", DockerVersionURL, err)
	}
	defer resp.Body.Close()

	// Get the version number alone.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version response body: %v", err)
	}
	versionString := strings.SplitN(string(body), "-", 2)[0]
	return versionString, nil
}
