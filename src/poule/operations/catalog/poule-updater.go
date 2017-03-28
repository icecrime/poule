package catalog

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"poule/configuration"
	"poule/gh"
	"poule/operations"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	pouleValidationCommentToken = "AUTOMATED:POULE:POULE-VALIDATION"
	pouleValidationContext      = "poule-validation"
)

// PouleUpdateCallback is the callback to call when a configuration update is required.
//
// OK, global state is terrible, but I like to think of `pouleUpdaterOperation` as an exception
// rather than the norm. If more of such operations need to exist in the future, we may want to
// create a special kind of "core operations" which have privileged access to the configuration.
var PouleUpdateCallback func(repository string) error

func init() {
	registerOperation(&pouleUpdaterDescriptor{})
}

type pouleUpdaterDescriptor struct{}

func (d *pouleUpdaterDescriptor) CommandLineDescription() CommandLineDescription {
	return CommandLineDescription{
		Name:        "poule-updater",
		Description: "Update the poule configuration for the specified repository",
	}
}

func (d *pouleUpdaterDescriptor) OperationFromCli(c *cli.Context) (operations.Operation, error) {
	return nil, fmt.Errorf("The poule-updater operation cannot be created from the command line")
}

func (d *pouleUpdaterDescriptor) OperationFromConfig(c operations.Configuration) (operations.Operation, error) {
	return &pouleUpdaterOperation{}, nil
}

type pouleUpdaterOperation struct{}

type pouleUpdaterUserData struct {
	Merged bool
	URL    string
}

func (o *pouleUpdaterOperation) Accepts() operations.AcceptedType {
	return operations.PullRequests
}

func (o *pouleUpdaterOperation) Apply(c *operations.Context, item gh.Item, userData interface{}) error {
	ud := userData.(pouleUpdaterUserData)
	if ud.Merged {
		return updatePouleConfiguration(item.Repository())
	}
	return validatePouleConfiguration(c, item, ud)
}

func (o *pouleUpdaterOperation) Describe(c *operations.Context, item gh.Item, userData interface{}) string {
	if ud := userData.(pouleUpdaterUserData); ud.Merged {
		return fmt.Sprintf("updating from merged configuration")
	}
	return fmt.Sprintf("validating unmerged configuration")
}

func (o *pouleUpdaterOperation) Filter(c *operations.Context, item gh.Item) (operations.FilterResult, interface{}, error) {
	// Exclude closed and unmerged pull requests.
	pr := item.PullRequest
	isMerged := pr.Merged != nil && *pr.Merged
	if *pr.State != "open" && !isMerged {
		logrus.Debugf("rejecting unnmerged pull request")
		return operations.Reject, nil, nil
	}

	// List all files modified by the pull requests, and look for our special configuration file.
	commitFiles, _, err := c.Client.PullRequests().ListFiles(c.Username, c.Repository, item.Number(), nil)
	if err != nil {
		return operations.Reject, nil, err
	}
	for _, commitFile := range commitFiles {
		if *commitFile.Filename == configuration.PouleConfigurationFile {
			userData := pouleUpdaterUserData{
				Merged: isMerged,
				URL:    *commitFile.RawURL,
			}
			return operations.Accept, userData, nil
		}
	}
	return operations.Reject, nil, nil
}

func (o *pouleUpdaterOperation) IssueListOptions(c *operations.Context) *github.IssueListByRepoOptions {
	return nil
}

func (o *pouleUpdaterOperation) PullRequestListOptions(c *operations.Context) *github.PullRequestListOptions {
	return nil
}

func updatePouleConfiguration(repository string) error {
	if PouleUpdateCallback == nil {
		return errors.New("poule configuration update callback is nil")
	}
	return PouleUpdateCallback(repository)
}

func validatePouleConfiguration(c *operations.Context, item gh.Item, userData pouleUpdaterUserData) error {
	resp, err := http.Get(userData.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error retrieving file %q (%s)", userData.URL, resp.Status)
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var actions configuration.Actions
	if err := yaml.Unmarshal(content, &actions); err != nil {
		return applyInvalidPouleConfiguration(c, item, userData, []error{err})
	} else if errs := actions.Validate(OperationValidator{}); len(errs) != 0 {
		return applyInvalidPouleConfiguration(c, item, userData, errs)
	}
	return applyValidPouleConfiguration(c, item, userData)
}

func applyInvalidPouleConfiguration(c *operations.Context, item gh.Item, userData pouleUpdaterUserData, errs []error) error {
	// Create the automated comment for that pull request, unless there is already one.
	pr := item.PullRequest
	if automatedComments, err := findAutomatedComments(c, pr, pouleValidationCommentToken); err != nil {
		return err
	} else if len(automatedComments) != 0 {
		return nil
	}

	// Create the automated comment.
	content := formatValidationComment(c, pr, errs)
	comment := &github.IssueComment{Body: &content}
	if _, _, err := c.Client.Issues().CreateComment(c.Username, c.Repository, *pr.Number, comment); err != nil {
		return err
	}

	_, _, err := c.Client.Repositories().CreateStatus(c.Username, c.Repository, *pr.Head.SHA, &github.RepoStatus{
		Context:     github.String(pouleValidationContext),
		Description: github.String("Poule configuration is invalid"),
		State:       github.String("failure"),
	})
	return err
}

func applyValidPouleConfiguration(c *operations.Context, item gh.Item, userData pouleUpdaterUserData) error {
	// Delete the automated validation comment (if any).
	pr := item.PullRequest
	if err := deleteAutomatedComments(c, pr, pouleValidationCommentToken); err != nil {
		return err
	}

	_, _, err := c.Client.Repositories().CreateStatus(c.Username, c.Repository, *pr.Head.SHA, &github.RepoStatus{
		Context:     github.String(pouleValidationContext),
		Description: github.String("Poule configuration is valid"),
		State:       github.String("success"),
	})
	return err
}

func formatValidationComment(c *operations.Context, pr *github.PullRequest, errs []error) string {
	var strErrors []string
	for _, err := range errs {
		strErrors = append(strErrors, err.Error())
	}

	comment := fmt.Sprintf("<!-- %s -->\n", pouleValidationCommentToken)
	comment += fmt.Sprintf(":chicken: Validation failed:\n```\n%s\n```\n", strings.Join(strErrors, "\n"))
	return comment
}
