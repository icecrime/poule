package catalog

import (
	"sort"

	"poule/operations"

	"github.com/urfave/cli"
)

// CommandLineDescription describes the command-line interface for an
// operation.
type CommandLineDescription struct {
	// Name is the operation's command.
	Name string

	// Description is the operation's help message.
	Description string

	// Flags is an array of operation-specific command line flags.
	Flags []cli.Flag

	// ArgsUsage describes the arguments to this command.
	ArgsUsage string
}

// OperationDescriptor describes an operation.
type OperationDescriptor interface {
	// CommandLineDescription returns the necessary information to populate the
	// command line with that operation.
	CommandLineDescription() CommandLineDescription

	// OperationFromCli returns a new instance of that operations configured as
	// described by command line flags and arguemnts.
	OperationFromCli(*cli.Context) (operations.Operation, error)

	// OperationFromConfig returns a new instance of that operation configured
	// as described by the opaque `operations.Configuration` structure.
	OperationFromConfig(operations.Configuration) (operations.Operation, error)
}

type OperationDescriptors []OperationDescriptor

func (d OperationDescriptors) Len() int {
	return len(d)
}

func (d OperationDescriptors) Less(i, j int) bool {
	return d[i].CommandLineDescription().Name < d[j].CommandLineDescription().Name
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
	ByNameIndex[descriptor.CommandLineDescription().Name] = descriptor
	sort.Sort(Index)
}
