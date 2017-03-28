package configuration

// Server is the configuration object for the server mode.
type Server struct {
	Config      `yaml:",inline"`
	LookupdAddr string `yaml:"nsq_lookupd"`
	Channel     string `yaml:"nsq_channel"`

	// Repositories maps GitHub repositories full names their corresponding
	// NSQ topic.
	Repositories map[string]string `yaml:"repositories"`

	// CommonActions defines the triggers and operations which apply to every configured repository.
	CommonActions []Action `yaml:"common_configuration"`
}

// Validate verifies the validity of the configuration.
func (s Server) Validate(opValidator OperationValidator) []error {
	var errs []error
	for _, action := range s.CommonActions {
		if err := action.Validate(opValidator); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
