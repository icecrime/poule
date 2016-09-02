package catalog

import "github.com/codegangsta/cli"

// Operation is an empty interface to encompass both issue and pull request
// operations in a single descriptor type.
type Operation interface{}

// OperationDescriptor describes an operation.
type OperationDescriptor interface {
	// Name is a short-name for the operation.
	Name() string

	// Command returns a CLI command to invoke the operation.
	Command() cli.Command

	// Operation returns a new instance of that operation.
	Operation() Operation
}

// Index is the catalog of all known operations by name.
var Index = map[string]OperationDescriptor{}

func registerOperation(descriptor OperationDescriptor) {
	Index[descriptor.Name()] = descriptor
}
