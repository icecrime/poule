package configuration

import (
	"fmt"
	"strings"
)

// Trigger associates a GitHub event type (e.g., "issues", or "pull request") with a collection of
// corresponding actions (e.g., [ "opened", "reopened" ]).
type Trigger map[string]StringSlice

// Contains returns whether the triggers contains the specified (event, action) tuple.
func (t Trigger) Contains(githubEvent, githubAction string) bool {
	if actions, ok := t[githubEvent]; ok {
		return actions.Contains(githubAction)
	}
	return false
}

// Validate verifies the validity of the trigger definition.
func (t Trigger) Validate() error {
	var invalidEvents []string
	for event, _ := range t {
		if !StringSlice(GitHubEvents).Contains(event) {
			invalidEvents = append(invalidEvents, fmt.Sprintf("%q", event))
		}
	}
	if len(invalidEvents) != 0 {
		return fmt.Errorf("Invalid event type %s", strings.Join(invalidEvents, ", "))
	}
	return nil
}
