package catalog

import (
	"poule/configuration"
	"poule/operations"

	"github.com/pkg/errors"
)

// OperationFromConfig returns an operation parsed from the configuration.
func OperationFromConfig(operationConfig *configuration.OperationConfiguration) (operations.Operation, error) {
	// Create the operation.
	descriptor, ok := ByNameIndex[operationConfig.Type]
	if !ok {
		return nil, errors.Errorf("unknown operation %q", operationConfig.Type)
	}
	operation, err := descriptor.OperationFromConfig(operationConfig.Settings)
	if err != nil {
		return nil, err
	}
	return operation, nil
}

// OperationValidator validates an operation configuration.
type OperationValidator struct{}

func (o OperationValidator) Validate(operationConfig *configuration.OperationConfiguration) error {
	_, err := OperationFromConfig(operationConfig)
	return err
}
