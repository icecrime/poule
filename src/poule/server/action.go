package server

import (
	"poule/configuration"
	"poule/gh"
	"poule/runner"

	"github.com/Sirupsen/logrus"
)

func executeAction(config *configuration.Config, action configuration.Action, item gh.Item) error {
	for _, opConfig := range action.Operations {
		logrus.WithFields(logrus.Fields{
			"operation":  opConfig.Type,
			"number":     item.Number(),
			"repository": item.Repository(),
		}).Info("running operation")

		opRunner, err := runner.NewOperationRunnerFromConfig(config, &opConfig)
		if err != nil {
			return err
		}
		if err := opRunner.Handle(item); err != nil {
			return err
		}
	}
	return nil
}

func executeActionOnAllItems(config *configuration.Config, action configuration.Action) error {
	for _, opConfig := range action.Operations {
		logrus.WithFields(logrus.Fields{
			"operation": opConfig.Type,
		}).Info("running operation on stock")

		opRunner, err := runner.NewOperationRunnerFromConfig(config, &opConfig)
		if err != nil {
			return err
		}
		if err := opRunner.HandleStock(); err != nil {
			return err
		}
	}
	return nil
}
