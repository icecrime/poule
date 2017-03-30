package listeners

// Handler handles GitHub events.
type Handler interface {
	// HandleMessage handles a GitHub event.
	HandleMessage(event string, body []byte) error
}

// Listener listens for GitHub events.
type Listener interface {
	// Start starts listening for GitHub events, calling the Handler for each event received.
	Start(handler Handler) error
}
