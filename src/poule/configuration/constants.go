package configuration

const (
	// JenkinsBaseURL is the base URL for the Jenkins CI server.
	JenkinsBaseURL = "https://leeroy.dockerproject.org/build/retry"

	// FailingCILabel is the label that indicates that a pull request is
	// failing for a legitimate reason and should be ignored.
	FailingCILabel = "status/failing-ci"

	// PouleToken is injected as an HTML comment in the body of all messages
	// posted by the tool itself.
	PouleToken = "AUTOMATED:POULE"

	// PouleConfigurationFile is the name of the special file at the root of the repository.
	PouleConfigurationFile = "poule.yml"
)
