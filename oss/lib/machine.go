package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Scoped to a bridge invocation, like the legacy command singletons. Concurrent
// invocations are unsupported; confirmation calls from transfer workers are safe.
type machineInvocation struct {
	format, cursor, phase, command string
	ai, nonInteractive             bool
	preview                        bool
	failures                       *failureManifest
	writer                         io.Writer
	confirmation                   atomic.Bool
	secrets                        []string
}

var activeMachine *machineInvocation

func addMachineFlags(fs *cli.FlagSet) {
	for _, f := range []struct {
		name, help string
		mode       cli.AssignedMode
	}{
		{"cli-failure-report", "Write an exclusive JSONL failed-transfer manifest for cp/sync; never replays writes.", cli.AssignedOnce},
		{"cli-validate", "Validate supported single-file cp / single-object rm locally; emits JSON, never executes.", cli.AssignedNone},
		{"cli-plan", "Read-only plan for supported single-file cp / single-object rm; emits JSON, never executes.", cli.AssignedNone},
		{"cli-ai-mode", "Return structured OSS errors; never grants confirmation.", cli.AssignedNone},
		{"no-cli-ai-mode", "Disable AI errors (explicit JSON output still uses JSON errors).", cli.AssignedNone},
		{"cli-output", "OSS output: text, json or jsonl. JSON lists are bounded; cat remains raw.", cli.AssignedOnce},
		{"cli-non-interactive", "Never read confirmation input; return ConfirmationRequired when needed.", cli.AssignedNone},
		{"cli-cursor", "Resume an OSS JSON/JSONL list with its opaque next_cursor.", cli.AssignedOnce},
	} {
		if fs.Get(f.name) == nil {
			fs.Add(&cli.Flag{Name: f.name, Short: i18n.T(f.help, f.help), AssignedMode: f.mode, Persistent: true})
		}
	}
}

func machineFromContext(ctx *cli.Context) *machineInvocation {
	assigned := func(name string) bool { f := ctx.Flags().Get(name); return f != nil && f.IsAssigned() }
	value := func(name string) string { v, _ := ctx.Flags().GetValue(name); return v }
	cfg, _ := aimode.Load(config.GetConfigDir(ctx))
	return &machineInvocation{format: value("cli-output"), cursor: value("cli-cursor"),
		preview:        assigned("cli-validate") || assigned("cli-plan"),
		ai:             aimode.EnabledForCommand(cfg, assigned("cli-ai-mode"), assigned("no-cli-ai-mode")),
		nonInteractive: assigned("cli-non-interactive"), writer: ctx.Stdout(), phase: "validation", command: ctx.Command().Name}
}
func (m *machineInvocation) structured() bool {
	return m.ai || m.format == "json" || m.format == "jsonl"
}
func (m *machineInvocation) validate(options OptionMapType) error {
	if m.format != "" && m.format != "text" && m.format != "json" && m.format != "jsonl" {
		return fmt.Errorf("--cli-output must be text, json or jsonl")
	}
	if m.cursor != "" && (m.command != "ls" || (m.format != "json" && m.format != "jsonl")) {
		return fmt.Errorf("--cli-cursor requires oss ls --cli-output json or jsonl")
	}
	if (m.format == "json" || m.format == "jsonl") && m.command != "ls" && m.command != "cat" && !m.preview {
		return fmt.Errorf("oss %s does not support JSON/JSONL success output", m.command)
	}
	if m.command == "config" && (m.nonInteractive || m.structured()) {
		return fmt.Errorf("oss config is interactive; configure the host CLI separately")
	}
	if m.command == "ls" && (m.format == "json" || m.format == "jsonl") {
		n, _ := GetInt(OptionLimitedNum, options)
		raw, _ := GetString(OptionLimitedNum, options)
		if raw != "" && (n < 1 || n > 1000) {
			return fmt.Errorf("machine lists require --limited-num between 1 and 1000")
		}
	}
	return nil
}

// This wrapper covers legacy confirmation and value prompts without treating AI
// mode as authorization. Human terminal behavior remains unchanged.
func scanOSSInput(a ...interface{}) (int, error) {
	if activeMachine != nil && (activeMachine.nonInteractive || activeMachine.structured()) {
		activeMachine.confirmation.Store(true)
		return 0, errConfirmationRequired
	}
	return fmt.Scanln(a...)
}

var errConfirmationRequired = errors.New("confirmation required; use an interactive terminal or explicitly authorize the operation with its existing options")

type ossErrorFacts struct {
	FailureReport string `json:"failure_report,omitempty"`
	DeletePhase   string `json:"delete_phase,omitempty"`
	SchemaVersion string `json:"schema_version"`
	Phase         string `json:"phase"`
	Status        string `json:"status"`
	SideEffects   string `json:"side_effects"`
	Retryable     bool   `json:"retryable"`
}
type ossAgentError struct {
	*cli.AgentError
	facts ossErrorFacts
}

func (e *ossAgentError) ExitCode() int { return 1 }
func (e *ossAgentError) RenderError(w io.Writer) error {
	return json.NewEncoder(w).Encode(struct {
		cli.AgentErrorEnvelope
		OSS ossErrorFacts `json:"oss"`
	}{e.Envelope(), e.facts})
}
func (m *machineInvocation) adaptError(err error) error {
	if err == nil {
		return nil
	}
	message := redactOSSDiagnostic(err.Error())
	for _, secret := range m.secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[REDACTED]")
		}
	}
	// Sanitize each URL independently, including URLs inside transport messages.
	for _, word := range strings.Fields(message) {
		clean := strings.Trim(word, "\"'(),")
		if strings.Contains(clean, "://") {
			message = strings.ReplaceAll(message, clean, redactSignedURL(clean))
		}
	}
	if !m.structured() {
		if message != err.Error() {
			return &sanitizedOSSError{err, message}
		}
		return err
	}
	facts := ossErrorFacts{SchemaVersion: "1", Phase: m.phase, Status: "failed", SideEffects: "unknown"}
	if m.phase != "execution" || m.command == "ls" || m.command == "cat" || m.command == "hash" {
		facts.SideEffects = "none"
	}
	if m.failures != nil {
		facts.FailureReport = m.failures.file.Name()
		facts.DeletePhase = m.failures.deletePhase
	}
	envelope := cli.AgentErrorEnvelope{Message: message, Recovery: cli.AgentErrorRecovery{Action: "inspect_error", Hint: "Inspect the error and operation state before retrying; do not replay writes automatically."}}
	if m.phase == "validation" {
		envelope.ErrorCode = "InvalidArgument"
		envelope.Recovery = cli.AgentErrorRecovery{Action: "check_arguments", Hint: "Check the command arguments and supported options with aliyun oss " + m.command + " --help."}
	}
	if m.phase == "configuration" {
		envelope.ErrorCode = "ConfigurationError"
	}
	if errors.Is(err, errConfirmationRequired) {
		envelope.ErrorCode = "ConfirmationRequired"
		facts.Status = "confirmation_required"
		envelope.Recovery = cli.AgentErrorRecovery{Action: "request_confirmation", Hint: "Obtain user authorization before using the command's existing force option. Earlier work may already have completed."}
	}
	var service oss.ServiceError
	serviceFound := errors.As(err, &service)
	if !serviceFound {
		var ptr *oss.ServiceError
		if errors.As(err, &ptr) && ptr != nil {
			service = *ptr
			serviceFound = true
		}
	}
	if serviceFound && !errors.Is(err, errConfirmationRequired) {
		envelope.ErrorCode = service.Code
		envelope.StatusCode = service.StatusCode
		envelope.RequestId = service.RequestID
		switch service.Code {
		case "AccessDenied":
			if service.Ec == "0003-00001403" || strings.Contains(strings.ToLower(service.Message), "must be addressed using the specified endpoint") {
				envelope.Recovery = cli.AgentErrorRecovery{Action: "check_endpoint", Hint: "Verify the bucket region, endpoint and signing region; do not guess a replacement or disable TLS."}
			} else {
				envelope.Recovery = cli.AgentErrorRecovery{Action: "check_permissions", Hint: "Check the selected identity, resource policy and required OSS action using the request ID; also verify that the endpoint and signing region match the bucket region. Do not broaden permissions automatically."}
			}
		case "NoSuchKey", "NoSuchBucket", "NoSuchVersion":
			envelope.Recovery = cli.AgentErrorRecovery{Action: "check_resource", Hint: "Verify the bucket, exact key and version ID before retrying."}
		case "InvalidAccessKeyId", "SecurityTokenExpired", "InvalidSecurityToken":
			envelope.Recovery = cli.AgentErrorRecovery{Action: "refresh_credentials", Hint: "Refresh credentials for the selected profile; verify prior writes before retrying."}
		case "PermanentRedirect", "AuthorizationHeaderMalformed":
			envelope.Recovery = cli.AgentErrorRecovery{Action: "check_endpoint", Hint: "Verify the bucket region, endpoint and signing region; do not guess a replacement or disable TLS."}
		}
		if service.StatusCode == 429 || service.StatusCode >= 500 {
			facts.Retryable = m.command == "ls" || m.command == "cat"
			envelope.Recovery = cli.AgentErrorRecovery{Action: "check_retry_safety", Hint: "Back off for transient failures. Reads may be retried; check the outcome of writes before replaying them."}
		}
	}
	var network net.Error
	if errors.As(err, &network) && !errors.Is(err, errConfirmationRequired) {
		envelope.Recovery = cli.AgentErrorRecovery{Action: "check_connection_and_state", Hint: "Check connectivity. A timed-out write may have completed; inspect its state before retrying."}
		facts.Retryable = m.command == "ls" || m.command == "cat"
	}
	if facts.FailureReport != "" {
		envelope.Recovery = cli.AgentErrorRecovery{Action: "inspect_failed_items", Hint: "Inspect the failure report and operation state; retry only reviewed failed items, never replay the entire sync/delete operation automatically."}
	}
	agentErr := cli.NewAgentError(envelope, err)
	if m.ai {
		agentErr = agentErr.WithSchemaVersion()
	}
	return &ossAgentError{agentErr, facts}
}

func machineInputBlocked() bool {
	return activeMachine != nil && (activeMachine.nonInteractive || activeMachine.structured())
}
