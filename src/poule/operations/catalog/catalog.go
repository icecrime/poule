package catalog

import (
	"sort"

	"poule/operations"

	"github.com/urfave/cli"
)

// Operation is an empty interface to encompass both issue and pull request
// operations in a single descriptor type.
type Operation interface{}

// OperationDescriptor describes an operation.
type OperationDescriptor interface {
	// Name is a short-name for the operation.
	Name() string

	// Description returns as help message for the operation.
	Description() string

	// OperationFromCli returns a new instance of that operations configured as
	// described by command line flags and arguemnts.
	OperationFromCli(*cli.Context) Operation

	// OperationFromConfig returns a new instance of that operation configured
	// as described by the opaque `operations.Configuration` structure.
	OperationFromConfig(operations.Configuration) Operation
}

type OperationDescriptors []OperationDescriptor

func (d OperationDescriptors) Len() int {
	return len(d)
}

func (d OperationDescriptors) Less(i, j int) bool {
	return d[i].Name() < d[j].Name()
}

func (d OperationDescriptors) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Index is the catalog of all known operations by name.
var (
	Index       OperationDescriptors
	ByNameIndex = map[string]OperationDescriptor{}
)

func registerOperation(descriptor OperationDescriptor) {
	Index = append(Index, descriptor)
	ByNameIndex[descriptor.Name()] = descriptor
	sort.Sort(Index)
}
