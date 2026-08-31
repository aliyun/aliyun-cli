// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package runtimehost is the aliyun-cli host adapter for the standalone aliyun-openapi-runtime engine.
// It is the ONLY place that binds the engine to aliyun-cli's cli.Command tree, config/profile credentials, and i18n, keeping the engine module itself free of those dependencies.
package runtimehost

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-cli/v3/bundledmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/headers"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
	"github.com/aliyun/aliyun-cli/v3/util"
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

// userPluginsDir resolves the directory holding user-installed meta plugins, matching the plugin manager's convention:
//
//	$ALIBABA_CLOUD_CLI_PLUGINS_DIR, or ~/.aliyun/plugins
//
// Returns "" when no home can be determined; the engine then simply omits the user layer.
func userPluginsDir() string {
	if d := os.Getenv(sysconfig.EnvPluginsDir); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".aliyun", "plugins")
}

// cliHost adapts aliyun-cli's profile, flags and system configuration to the engine's runtime.Host seam.
// The profile is loaded lazily and once, so dry-run (which only reads Region) never triggers credential IO.
type cliHost struct {
	ctx     *cli.Context
	once    sync.Once
	profile config.Profile
	loadErr error
}

func (h *cliHost) load() {
	h.once.Do(func() {
		h.profile, h.loadErr = config.LoadProfileWithContext(h.ctx)
	})
}

// Region returns the profile's default region, or "" if the profile
// cannot be loaded (dry-run tolerates this and resolves region-less).
func (h *cliHost) Region() string {
	h.load()
	if h.loadErr != nil {
		return ""
	}
	return h.profile.RegionId
}

// Settings returns the profile-derived wire settings. Best-effort: on
// a load error it returns zero values (the engine then uses SDK
// defaults). Timeouts in the profile are seconds; convert to Duration.
// Transport flags (--skip-secure-verify, --user-agent, AI mode) are
// folded in from the flags already parsed into ctx.
func (h *cliHost) Settings() runtime.Settings {
	h.load()
	s := runtime.Settings{}
	if h.loadErr == nil {
		s.ReadTimeout = time.Duration(h.profile.ReadTimeout) * time.Second
		s.ConnectTimeout = time.Duration(h.profile.ConnectTimeout) * time.Second
		s.RetryCount = h.profile.RetryCount
		s.EndpointType = h.profile.EndpointType
		s.Language = h.profile.Language
	}
	s.CLIVersion = cli.GetVersion()
	if h.ctx != nil {
		if f := h.ctx.Flags().Get(config.SkipSecureVerifyName); f != nil && f.IsAssigned() {
			s.SkipSecureVerify = true
		}
		s.UserAgent = buildUserAgentSuffix(h.ctx)
	}
	return s
}

// TransportOptions resolves non-secret request context without touching credentials.
// Configuration parsing stays in the aliyun-cli host adapter.
func (h *cliHost) TransportOptions() runtime.TransportOptions {
	options := runtime.TransportOptions{
		Headers: headers.Collect(),
		CallContext: runtime.CallContextOptions{
			SourceIP:        strings.TrimSpace(os.Getenv(sysconfig.EnvSourceIP)),
			SecureTransport: strings.TrimSpace(os.Getenv(sysconfig.EnvSecureTransport)),
			SkipProducts:    splitCommaList(os.Getenv(sysconfig.EnvCallContextSkipProducts)),
		},
	}
	cfg, err := throttlingretry.LoadEffective(config.GetConfigDir(h.ctx))
	if err == nil && cfg != nil {
		options.ThrottlingRetry = runtime.ThrottlingRetryOptions{
			Enabled:     cfg.Enabled,
			MaxAttempts: cfg.MaxAttempts,
			MaxDelayMS:  cfg.MaxDelayMS,
		}
	}
	return options
}

func splitCommaList(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

// buildUserAgentSuffix merges ALIBABA_CLOUD_USER_AGENT and --user-agent
// with the host AI-mode suffix.
// (config + detected agent + --cli-ai-mode / --no-cli-ai-mode).
// The engine prefixes Aliyun-CLI/{cliVer} aliyun-openapi-runtime/{ver}.
func buildUserAgentSuffix(ctx *cli.Context) string {
	var parts []string
	if value := strings.TrimSpace(os.Getenv(sysconfig.EnvUserAgent)); value != "" {
		if value = strings.TrimSpace(util.SanitizeUserAgent(value)); value != "" {
			parts = append(parts, value)
		}
	}
	if f := ctx.Flags().Get("user-agent"); f != nil {
		if v, ok := f.GetValue(); ok && strings.TrimSpace(v) != "" {
			if v = strings.TrimSpace(util.SanitizeUserAgent(v)); v != "" {
				parts = append(parts, v)
			}
		}
	}

	cfg, enabled := aiModeForCommand(ctx)
	if enabled {
		forceOn := flagAssigned(ctx, "cli-ai-mode")
		forceOff := flagAssigned(ctx, "no-cli-ai-mode")
		detectedAgent := ctx != nil && ctx.IsAgent()
		suf := aimode.RequestUserAgentSuffixForCommandWithDetectedAgent(
			cfg, forceOn, forceOff, detectedAgent,
		)
		parts = append(parts, suf)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func aiModeForCommand(ctx *cli.Context) (*aimode.AiConfig, bool) {
	cfg, err := aimode.Load(config.GetConfigDir(ctx))
	if err != nil {
		cfg = aimode.DefaultAiConfig()
	}
	forceOn := flagAssigned(ctx, "cli-ai-mode")
	forceOff := flagAssigned(ctx, "no-cli-ai-mode")
	if !forceOff && ctx != nil && ctx.IsAgent() {
		forceOn = true
	}
	return cfg, aimode.EnabledForCommand(cfg, forceOn, forceOff)
}

func flagAssigned(ctx *cli.Context, name string) bool {
	if ctx == nil {
		return false
	}
	f := ctx.Flags().Get(name)
	return f != nil && f.IsAssigned()
}

func checkSafetyPolicy(ctx *cli.Context, rawArgs []string) error {
	if ctx == nil || len(rawArgs) < 2 {
		return nil
	}

	// Metadata commands are interpreted in-process, so they never reach the Go-plugin safety check in openapi.Commando.
	// Keep read-only help/version requests aligned with that path and guard every actual metadata command here,
	// where both bundled baseline and installed meta plugins converge.
	command := rawArgs[1]
	if strings.EqualFold(command, "version") || flagAssigned(ctx, "help") {
		return nil
	}
	for _, arg := range rawArgs[2:] {
		if arg == "--help" || arg == "-h" {
			return nil
		}
	}

	policy, err := safety.LoadEffectivePolicy(config.GetConfigDir(ctx))
	if err != nil {
		// Preserve the existing host policy semantics: malformed/unreadable
		// policy configuration fails open rather than breaking CLI execution.
		return nil
	}
	skipConfirm := flagAssigned(ctx, "yes") ||
		os.Getenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM") == "1" ||
		strings.EqualFold(os.Getenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM"), "true")
	return safety.CheckAndConfirm(ctx, policy, safety.CommandInfo{
		Product:     rawArgs[0],
		ApiOrMethod: command,
	}, skipConfirm)
}

// Credential resolves the caller's credential from the profile.
func (h *cliHost) Credential() (credentialsv2.Credential, error) {
	h.load()
	if h.loadErr != nil {
		return nil, h.loadErr
	}
	return h.profile.GetCredential(h.ctx, nil)
}

// sharedEngine is the process-wide engine, built once. The top-level product router dispatches through it and resolves only the requested product.
var (
	engineOnce     sync.Once
	engineInst     *engine.Engine
	engineDispatch = func(request engine.Request) error {
		return Engine().Dispatch(request)
	}
)

func Engine() *engine.Engine {
	engineOnce.Do(func() {
		engineInst = openapiruntime.NewEngine(openapiruntime.Options{
			BaselineFS:     bundledmeta.Metadatas,
			BundledBy:      "aliyun-cli " + cli.Version,
			UserPluginsDir: userPluginsDir(),
			OverrideDir:    os.Getenv("ALIYUN_CLI_PLUGINS_DIR_OVERRIDE"),
			ExternalFlags:  engineExternalFlags(),
		}, nil)
	})
	return engineInst
}

// Dispatch runs "<product> <command> [--flags...]" through the engine
// using the profile-backed host. rawArgs is the argv tail starting at
// the product (e.g. os.Args from the product onward).
//
// The root parser has already stored host global flags (profile /
// credential / network) in ctx. Their raw argv tokens are declared as
// external flags so the engine parser can consume and ignore them.
func Dispatch(ctx *cli.Context, rawArgs []string) error {
	if err := checkSafetyPolicy(ctx, rawArgs); err != nil {
		return err
	}
	host := &cliHost{ctx: ctx}

	// Resolve the display language from the profile (which now reflects
	// a command-line --language via OverwriteWithFlags), falling back to
	// the process language. Also push it into i18n so host-side messages
	// stay consistent.
	lang := host.Settings().Language
	if lang == "" {
		lang = i18n.GetLanguage()
	} else {
		i18n.SetLanguage(lang)
	}
	_, aiModeEnabled := aiModeForCommand(ctx)

	return engineDispatch(engine.Request{
		Args:   rawArgs,
		Out:    ctx.Stdout(),
		Lang:   lang,
		Host:   host,
		AIMode: aiModeEnabled,
	})
}

// DispatchPluginHelp forwards an installed metadata plugin Help invocation
// without safety policy, credential loading, argv rebuilding, or modifier
// loss. Runtime Help receives the host's effective AI mode and language.
func DispatchPluginHelp(ctx *cli.Context, rawArgs []string) error {
	if handled, err := TryHelp(ctx, rawArgs); handled {
		return err
	}
	return nil
}

func helpLanguage(ctx *cli.Context) string {
	if ctx != nil && ctx.Flags() != nil {
		if f := config.LanguageFlag(ctx.Flags()); f != nil {
			if lang, ok := f.GetValue(); ok && strings.TrimSpace(lang) != "" {
				return strings.TrimSpace(lang)
			}
		}
	}
	return i18n.GetLanguage()
}

// ProductHelp renders a product-level kebab command list from the selected common-runtime source.
// For an installed metadata plugin the user source wins over baseline.
func ProductHelp(ctx *cli.Context, product string) error {
	args := []string{product}
	for i, arg := range os.Args {
		if arg == product {
			args = append([]string(nil), os.Args[i:]...)
			break
		}
	}
	_, err := TryHelp(ctx, args)
	return err
}

func HasProduct(product string) bool {
	return product != "" && Engine().HasProduct(product)
}

// ProductCommands returns the kebab command names the engine serves for a
// product, or nil when the product cannot be resolved. It is the candidate
// source for host-side suggestions on kebab commands.
func ProductCommands(product string) []string {
	if product == "" {
		return nil
	}
	return Engine().ProductCommands(product)
}

// MetaPluginProvenance resolves product ownership through the engine loader.
// A non-nil record whose Kind is not KindBaseline means an installed metadata
// plugin (user or override layer) owns the product and its data must be
// preferred over the bundled baseline.
func MetaPluginProvenance(product string) *source.Provenance {
	if product == "" {
		return nil
	}
	ldr, err := Engine().Loader()
	if err != nil {
		return nil
	}
	if err := ldr.EnsureProduct(product); err != nil {
		return nil
	}
	return ldr.Provenance(product)
}

// TryDispatch handles rawArgs via the engine only when the engine can resolve the "<product> <command>" pair.
// It returns handled=false (with nil error) otherwise,
// letting the caller fall back to legacy routing (auto-install, built-in openapi, ...).
func TryDispatch(ctx *cli.Context, rawArgs []string) (handled bool, err error) {
	if len(rawArgs) < 2 {
		return false, nil
	}
	if !Engine().Resolvable(rawArgs[0], rawArgs[1]) {
		return false, nil
	}
	return true, Dispatch(ctx, rawArgs)
}

// TryHelp routes a raw Help invocation to Runtime Help v1 when its target is
// a Runtime product or a resolvable lowercase kebab action. It preserves the
// original action tail so request, response and parameter Help share Runtime's
// parser and validation. Host Root, Utility and PascalCase targets are left to
// the caller.
func TryHelp(ctx *cli.Context, rawArgs []string) (handled bool, err error) {
	args, product, command := runtimeHelpArgs(rawArgs)
	if product == "" {
		return false, nil
	}
	if command == "" {
		if !Engine().HasProduct(product) {
			return false, nil
		}
		request := runtimeHelpRequest(ctx, rawArgs, args)
		return true, Engine().ProductHelp(request)
	}
	if strings.ToLower(command) != command || !Engine().Resolvable(product, command) {
		return false, nil
	}
	return true, Engine().APIHelp(runtimeHelpRequest(ctx, rawArgs, args))
}

func runtimeHelpRequest(ctx *cli.Context, rawArgs, args []string) engine.Request {
	lang := helpLanguage(ctx)
	if explicit := helpLanguageFromArgs(rawArgs); explicit != "" {
		lang = explicit
	}
	return engine.Request{
		Args:   args,
		Out:    ctx.Stdout(),
		Lang:   lang,
		AIMode: helpAIMode(ctx, rawArgs),
	}
}

func helpLanguageFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--language" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if strings.HasPrefix(arg, "--language=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--language="))
		}
	}
	return ""
}

func helpAIMode(ctx *cli.Context, args []string) bool {
	cfg, _ := aiModeForCommand(ctx)
	forceOn := flagAssigned(ctx, "cli-ai-mode")
	forceOff := flagAssigned(ctx, "no-cli-ai-mode")
	for _, arg := range args {
		switch strings.SplitN(arg, "=", 2)[0] {
		case "--cli-ai-mode":
			forceOn = true
		case "--no-cli-ai-mode":
			forceOff = true
		}
	}
	if !forceOff && ctx != nil && ctx.IsAgent() {
		forceOn = true
	}
	return aimode.EnabledForCommand(cfg, forceOn, forceOff)
}

func runtimeHelpArgs(rawArgs []string) (args []string, product, command string) {
	raw := append([]string(nil), rawArgs...)
	if len(raw) > 0 && raw[0] == "help" {
		raw = raw[1:]
	}
	productIndex := nextRuntimeHelpPositional(raw, 0)
	if productIndex < 0 {
		return nil, "", ""
	}
	product = raw[productIndex]
	commandIndex := nextRuntimeHelpPositional(raw, productIndex+1)
	if commandIndex < 0 {
		return append([]string{product}, runtimeHelpOptions(raw)...), product, ""
	}
	command = raw[commandIndex]
	args = []string{product, command}
	args = append(args, runtimeHelpOptions(raw[:commandIndex])...)
	args = append(args, raw[commandIndex+1:]...)
	return args, product, command
}

func nextRuntimeHelpPositional(args []string, start int) int {
	for i := start; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}
		name := strings.SplitN(arg, "=", 2)[0]
		if strings.HasPrefix(name, "-") {
			if runtimeHelpScanOptionTakesValue(name) && arg == name && i+1 < len(args) {
				i++
			}
			continue
		}
		return i
	}
	return -1
}

func runtimeHelpOptions(args []string) []string {
	result := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name := strings.SplitN(arg, "=", 2)[0]
		if !runtimeHelpOption(name) {
			continue
		}
		result = append(result, arg)
		if runtimeHelpOptionTakesValue(name) && arg == name && i+1 < len(args) {
			result = append(result, args[i+1])
			i++
		}
	}
	return result
}

func runtimeHelpOption(name string) bool {
	switch name {
	case "--help", "-h", "--api-version", "--language", "--cli-section",
		"--help-search", "--help-all", "--cli-output", "--cli-ai-mode",
		"--no-cli-ai-mode":
		return true
	}
	return false
}

func runtimeHelpOptionTakesValue(name string) bool {
	switch name {
	case "--api-version", "--language", "--cli-section", "--help-search", "--cli-output":
		return true
	}
	return false
}

func runtimeHelpScanOptionTakesValue(name string) bool {
	if runtimeHelpOptionTakesValue(name) {
		return true
	}
	for _, spec := range engineExternalFlags() {
		if name != "--"+spec.Name && (spec.Shorthand == 0 || name != "-"+string(spec.Shorthand)) {
			continue
		}
		return spec.Mode == argparser.ExternalFlagRequired || spec.Mode == argparser.ExternalFlagOptional
	}
	return false
}
