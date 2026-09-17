package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

func redactOSSValues(message string, secrets []string) string {
	for _, s := range secrets {
		if s != "" {
			message = strings.ReplaceAll(message, s, "[REDACTED]")
		}
	}
	for _, word := range strings.Fields(message) {
		clean := strings.Trim(word, "\"'(),")
		if strings.Contains(clean, "://") {
			message = strings.ReplaceAll(message, clean, redactSignedURL(clean))
		}
	}
	return message
}
func redactOSSDiagnostic(message string) string {
	if activeMachine != nil {
		message = redactOSSValues(message, activeMachine.secrets)
	}
	if bridgeCredentialProvider != nil {
		message = bridgeCredentialProvider.redact(message)
	}
	return redactOSSValues(message, nil)
}

type failureItem struct {
	SchemaVersion  string `json:"schema_version"`
	Type           string `json:"type"`
	Operation      string `json:"operation,omitempty"`
	Source         string `json:"source,omitempty"`
	Target         string `json:"target,omitempty"`
	VersionID      string `json:"version_id,omitempty"`
	Error          string `json:"error,omitempty"`
	Outcome        string `json:"outcome"`
	DeletePhase    string `json:"delete_phase,omitempty"`
	AutomaticRetry bool   `json:"automatic_retry"`
}
type failureManifest struct {
	deletePhase string
	mu          sync.Mutex
	file        *os.File
	err         error
}

func newFailureManifest(path string) (*failureManifest, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create failure manifest: %w", err)
	}
	return &failureManifest{file: f}, nil
}
func (m *failureManifest) write(item failureItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return
	}
	item.SchemaVersion = "1"
	item.Error = redactOSSDiagnostic(item.Error)
	m.err = json.NewEncoder(m.file).Encode(item)
}
func (m *failureManifest) finish(err error) error {
	item := failureItem{Type: "summary", Outcome: "succeeded", DeletePhase: m.deletePhase}
	if err != nil {
		item.Outcome = "failed"
		item.Error = err.Error()
	}
	m.write(item)
	return errors.Join(err, m.err, m.file.Close())
}
func recordTransferFailure(operation, source, target, version string, err error) {
	if err == nil || activeMachine == nil || activeMachine.failures == nil {
		return
	}
	activeMachine.failures.write(failureItem{Type: "failed_item", Operation: operation, Source: source, Target: target, VersionID: version, Error: err.Error(), Outcome: "unknown"})
}

type sanitizedOSSError struct {
	cause   error
	message string
}

func (e *sanitizedOSSError) Error() string { return e.message }
func (e *sanitizedOSSError) Unwrap() error { return e.cause }
