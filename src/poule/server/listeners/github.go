package listeners

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"poule/configuration"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// GitHubListener listens for GitHub events directly from webhooks.
type GitHubListener struct {
	config *configuration.Server
}

// NewGitHubListener returns a new GitHubListener instance.
func NewGitHubListener(config *configuration.Server) *GitHubListener {
	return &GitHubListener{
		config: config,
	}
}

// Start starts an HTTP server to receive GitHub WebHooks.
func (l *GitHubListener) Start(handler Handler) error {
	r := mux.NewRouter()
	r.Handle("/{user:.*}/{name:.*}", newWebHookHandler(handler, l.config.HTTPSecret)).Methods("POST")
	logrus.Infof("listening on %q", l.config.HTTPListen)
	return http.ListenAndServe(l.config.HTTPListen, r)
}

type webHookHandler struct {
	handler Handler
	secret  string
}

func newWebHookHandler(handler Handler, secret string) *webHookHandler {
	return &webHookHandler{
		handler: handler,
		secret:  secret,
	}
}

func (h *webHookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()

	if err != nil {
		logrus.WithField("error", err).Error("read request body")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !validateSignature(r, h.secret, data) {
		logrus.Warn("signature verification failed")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err := h.handler.HandleMessage(r.Header.Get("X-Github-Event"), data); err != nil {
		logrus.WithField("error", err).Error("processing event")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

/*
 * The helper function below courtesy of https://github.com/crosbymichael/hooks/
 */

// validateSignature validates the request payload with the user provided key using the
// HMAC algo
func validateSignature(r *http.Request, key string, payload []byte) bool {
	// if we don't have a secret to validate then just return true
	// because the user does not care about security
	if key == "" {
		return true
	}
	actual := r.Header.Get("X-Hub-Signature")
	expected, err := getExpectedSignature([]byte(key), payload)
	if err != nil {
		logrus.WithField("gh_signature", actual).WithField("error", err).Error("parse expected signature")
		return false
	}
	return hmac.Equal([]byte(expected), []byte(actual))
}

// getExpectedSignature returns the expected signature for the payload by
// applying the HMAC algo with sha1 as the digest to sign the request with
// the provided key
func getExpectedSignature(key, payload []byte) (string, error) {
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(payload); err != nil {
		return "", nil
	}
	return fmt.Sprintf("sha1=%s", hex.EncodeToString(mac.Sum(nil))), nil
}
