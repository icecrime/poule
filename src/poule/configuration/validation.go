package configuration

// OperationValidator validates an operation definition. We need an interface here to avoid a
// direct dependency on the operations package.
type OperationValidator interface {
	Validate(*OperationConfiguration) error
}
