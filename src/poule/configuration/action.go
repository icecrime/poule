package configuration

import (
	"fmt"

	cron "gopkg.in/robfig/cron.v2"
)

// Action is the definition of an action: it describes operations to apply when any of the
// associated triggers are met.
type Action struct {
	// Triggers is the collection of GitHub events that should trigger the action. The keys must be
	// valid GitHub event types (e.g., "pull_request"), and the value must be a list of valid values
	// for the action field of the GitHub paylost (e.g., "created").
	Triggers Trigger `yaml:"triggers"`

	// Schedule is a cron expression (https://godoc.org/gopkg.in/robfig/cron.v2).
	Schedule string `yaml:"schedule"`

	// Operations to apply to all repositories when any trigger is met.
	Operations []OperationConfiguration `yaml:"operations"`
}

// Validate verifies the validity of the action definition.
func (a Action) Validate(opValidator OperationValidator) error {
	if err := a.Triggers.Validate(); err != nil {
		return err
	}
	if a.Schedule != "" {
		if _, err := cron.Parse(a.Schedule); err != nil {
			return fmt.Errorf("Invalid schedule specification %q", a.Schedule)
		}
	}
	for _, opConfig := range a.Operations {
		if err := opValidator.Validate(&opConfig); err != nil {
			return err
		}
	}
	return nil
}

// Actions is a collection of Action.
type Actions []Action

// Validate verifies the validity of the configuration.
func (a Actions) Validate(opValidator OperationValidator) []error {
	var errs []error
	for _, action := range a {
		if err := action.Validate(opValidator); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
