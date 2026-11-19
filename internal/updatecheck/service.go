// Package updatecheck schedules cache-only failure notices and read-only checks.
package updatecheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/internal/background"
	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	cloudendpoint "github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"golang.org/x/mod/semver"
)

const (
	WorkerCommand  = "__update-check"
	checkInterval  = 24 * time.Hour
	remindInterval = 72 * time.Hour
	resultLifetime = 7 * 24 * time.Hour
	launchLease    = time.Minute
	checkTimeout   = 30 * time.Second
)

type Report struct {
	SchemaVersion int                  `json:"schema_version"`
	Status        string               `json:"status"`
	CheckedAt     time.Time            `json:"checked_at"`
	CLI           upgrade.UpdateResult `json:"cli"`
	Plugins       plugin.UpdateReport  `json:"plugins"`
}

type cache struct {
	AttemptAt     time.Time             `json:"attempt_at"`
	LaunchAt      time.Time             `json:"launch_at"`
	CLI           *upgrade.UpdateResult `json:"cli,omitempty"`
	CLIValidUntil time.Time             `json:"cli_valid_until"`
	Plugins       *plugin.UpdateReport  `json:"plugins,omitempty"`
	PluginProof   trust.CacheProof      `json:"plugin_proof"`
}

type Service struct {
	ConfigDir string
	Scope     plugin.UpdateScope
	Suffix    string
	Now       func() time.Time
	CLI       func(context.Context, string) (upgrade.UpdateResult, error)
	Plugins   func(context.Context, string, plugin.UpdateScope) (plugin.UpdateReport, error)
	Spawn     func(string, []string, []string) error
}

func New(dir string) (*Service, error) {
	if err := cloudendpoint.Validate(); err != nil {
		return nil, fmt.Errorf("invalid endpoint suffix setting")
	}
	scope, err := plugin.ResolveUpdateScope(dir)
	if err != nil {
		return nil, err
	}
	return &Service{ConfigDir: dir, Scope: scope, Suffix: strings.TrimSpace(os.Getenv(cloudendpoint.EnvSuffix)), Now: time.Now, CLI: upgrade.CheckUpdate, Plugins: plugin.CheckUpdates, Spawn: background.Start}, nil
}

func (s *Service) key() string {
	body, _ := json.Marshal([]string{s.Scope.Root, s.Scope.SourceBase, s.Scope.Site, s.Suffix})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:16])
}
func (s *Service) dir() string            { return filepath.Join(s.ConfigDir, "update-notifier-cache") }
func (s *Service) path(ext string) string { return filepath.Join(s.dir(), s.key()+ext) }

func readJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("update state too large")
	}
	return json.Unmarshal(data, value)
}
func writeJSON(path string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return localstate.Write(path, body)
}

// Check is explicit when force=true. Automatic callers must honor the interval.
func (s *Service) Check(ctx context.Context, force bool) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	r := Report{SchemaVersion: 1, Status: "failed", CheckedAt: s.Now().UTC()}
	if err := os.MkdirAll(s.dir(), 0700); err != nil {
		return r, fmt.Errorf("update cache is not writable")
	}
	// Only the worker/explicit query waits. The invoking business process uses
	// a nonblocking scheduling attempt and never waits for this lock.
	unlock, err := localstate.Lock(ctx, s.path(".check.lock"))
	if err != nil {
		return r, err
	}
	defer unlock()
	var state cache
	if err := readJSON(s.path(".json"), &state); err != nil {
		state = cache{}
	}
	if !force && !due(s.Now(), state.AttemptAt, checkInterval) {
		return r, nil
	}
	state.AttemptAt = s.Now().UTC()
	if err := writeJSON(s.path(".json"), state); err != nil {
		return r, fmt.Errorf("update cache cannot be saved")
	}
	var cliErr, pluginErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); r.CLI, cliErr = s.CLI(ctx, cli.GetVersion()) }()
	go func() { defer wg.Done(); r.Plugins, pluginErr = s.Plugins(ctx, s.ConfigDir, s.Scope) }()
	wg.Wait()
	if cliErr == nil {
		state.CLI, state.CLIValidUntil = &r.CLI, s.Now().Add(resultLifetime)
	}
	if pluginErr == nil || r.Plugins.Status == "cached" {
		state.Plugins, state.PluginProof = &r.Plugins, r.Plugins.Proof
	}
	if err := writeJSON(s.path(".json"), state); err != nil {
		return r, fmt.Errorf("update cache cannot be saved")
	}
	r.Status = "complete"
	if cliErr != nil || pluginErr != nil {
		r.Status = "partial"
		return r, errors.Join(cliErr, pluginErr)
	}
	return r, nil
}

func due(now, last time.Time, interval time.Duration) bool {
	return last.IsZero() || last.After(now.Add(5*time.Minute)) || !now.Before(last.Add(interval))
}

func Skip(args []string) bool {
	switch sysmock.FirstCommandToken(args) {
	case "upgrade", "plugin", "configure", "mock", "auto-completion", "mcp", "mcp-proxy", WorkerCommand, "__telemetry-upload":
		return true
	}
	return os.Getenv("GENERATE_METADATA") == "YES" || os.Getenv("COMP_LINE") != "" || os.Getenv("_ARGCOMPLETE") != ""
}

// Begin binds a cache snapshot without adding any successful-command output.
func Begin(dir string, ctx *cli.Context, args []string) {
	defer func() { _ = recover() }()
	if Skip(args) || !updatenotifier.Enabled(dir) || ctx.Completion() != nil {
		return
	}
	s, err := New(dir)
	if err != nil {
		return
	}
	var snapshot cache
	if err := readJSON(s.path(".json"), &snapshot); err != nil {
		snapshot = cache{}
	}
	ctx.SetUpdateNoticeProvider(s.provider(ctx, snapshot))
	// A Go test executable is not a CLI worker; never recursively launch it.
	if testing.Testing() {
		return
	}
	s.schedule()
}

func (s *Service) schedule() {
	if err := os.MkdirAll(s.dir(), 0700); err != nil {
		return
	}
	unlock, err := localstate.Lock(nil, s.path(".check.lock"))
	if err != nil {
		return
	}
	defer unlock()
	var state cache
	_ = readJSON(s.path(".json"), &state)
	if !due(s.Now(), state.AttemptAt, checkInterval) || !due(s.Now(), state.LaunchAt, launchLease) {
		return
	}
	state.LaunchAt = s.Now().UTC()
	if writeJSON(s.path(".json"), state) != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = s.Spawn(exe, []string{WorkerCommand, "--config-dir", s.ConfigDir, "--scope", s.key()}, os.Environ())
}

func RunWorker(args []string) int {
	// Errors are deliberately silent; the invoking business command already ran.
	dir, expected := "", ""
	if len(args) != 4 || args[0] != "--config-dir" || args[2] != "--scope" {
		return 0
	}
	dir, expected = args[1], args[3]
	if !filepath.IsAbs(dir) || !updatenotifier.Enabled(dir) {
		return 0
	}
	s, err := New(dir)
	if err != nil || s.key() != expected {
		return 0
	}
	_, _ = s.Check(context.Background(), false)
	return 0
}

func RunCheck(dir string, ctx *cli.Context) error {
	s, err := New(dir)
	if err != nil {
		return err
	}
	return s.runCheck(ctx)
}

func (s *Service) runCheck(ctx *cli.Context) error {
	r, checkErr := s.Check(context.Background(), true)
	if err := writeCheckReport(ctx, r); err != nil {
		return err
	}
	return checkErr
}

func newer(current, target string) bool {
	c, t := "v"+strings.TrimPrefix(current, "v"), "v"+strings.TrimPrefix(target, "v")
	return semver.IsValid(c) && semver.IsValid(t) && semver.Compare(c, t) < 0
}

func (s *Service) provider(ctx *cli.Context, snapshot cache) func() ([]cli.UpdateNotice, func(bool)) {
	used := false
	return func() ([]cli.UpdateNotice, func(bool)) {
		noop := func(bool) {}
		if used || !updatenotifier.Enabled(s.ConfigDir) {
			return nil, noop
		}
		used = true
		kind, owner := ctx.UpdateOwner()
		if kind == "passthrough" {
			return nil, noop
		}
		currentScope, err := New(s.ConfigDir)
		if err != nil || currentScope.key() != s.key() {
			return nil, noop
		}
		now := s.Now()
		var candidates []cli.UpdateNotice
		if r := snapshot.CLI; r != nil && r.Status != "development_build" && now.Before(snapshot.CLIValidUntil) && newer(cli.GetVersion(), r.Target) {
			candidates = append(candidates, cli.UpdateNotice{Component: "aliyun", Current: cli.GetVersion(), Latest: r.Target, Severity: "update", Command: "aliyun upgrade", Message: fmt.Sprintf("Alibaba Cloud CLI: %s → %s. Run aliyun upgrade.", cli.GetVersion(), r.Target)})
		}
		if snapshot.Plugins != nil && snapshot.PluginProof.Valid(now) && owner != "" {
			local, err := plugin.ReadInstalled(s.Scope.Root)
			if err == nil {
				for _, p := range snapshot.Plugins.Plugins {
					installed, ok := local.Plugins[p.Name]
					if !ok || p.Name != owner || !newer(installed.Version, p.Latest) || (p.Status != "update_available" && p.Status != "up_to_date") {
						continue
					}
					message := fmt.Sprintf("Plugin %s: %s → %s. Run %s.", p.Name, installed.Version, p.Latest, p.Command)
					if newer(cli.GetVersion(), strings.TrimSpace(p.MinCliVersion)) || newer(cli.GetVersion(), strings.TrimSpace(installed.MinCliVersion)) {
						message += " Upgrade the CLI first (aliyun upgrade)."
					}
					candidates = append(candidates, cli.UpdateNotice{Component: p.Name, Current: installed.Version, Latest: p.Latest, Severity: "update", Command: p.Command, Message: message})
				}
			}
		}
		if len(candidates) == 0 {
			return nil, noop
		}
		unlock, err := localstate.Lock(nil, s.path(".reminders.lock"))
		if err != nil {
			return nil, noop
		}
		history := map[string]time.Time{}
		if err := readJSON(s.path(".reminders.json"), &history); err != nil && !os.IsNotExist(err) {
			unlock()
			return nil, noop
		}
		if history == nil {
			history = map[string]time.Time{}
		}
		var notices []cli.UpdateNotice
		for _, n := range candidates {
			if due(now, history[n.Component+":"+n.Latest+":"+n.Severity], remindInterval) {
				notices = append(notices, n)
			}
		}
		var once sync.Once
		return notices, func(rendered bool) {
			once.Do(func() {
				defer unlock()
				if rendered && len(notices) > 0 {
					for _, n := range notices {
						history[n.Component+":"+n.Latest+":"+n.Severity] = now
					}
					_ = writeJSON(s.path(".reminders.json"), history)
				}
			})
		}
	}
}
