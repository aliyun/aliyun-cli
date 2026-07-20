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

// Package oapicmd is the aliyun-cli host adapter for the standalone
// aliyun-openapi-runtime engine. It is the ONLY place that binds the engine to
// aliyun-cli's cli.Command tree, config/profile credentials, and i18n,
// keeping the engine module itself free of those dependencies.
package oapicmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/v3/aliyun-openapi-meta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	openapiruntime "github.com/aliyun/aliyun-openapi-runtime"
	"github.com/aliyun/aliyun-openapi-runtime/jsoncmd"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

// envPluginsDir mirrors cli/plugin.EnvPluginsDir. Duplicated here (a
// single string) to avoid importing the whole plugin manager for one
// constant.
const envPluginsDir = "ALIBABA_CLOUD_CLI_PLUGINS_DIR"

// userPluginsDir resolves the directory holding user-installed meta
// plugins, matching the plugin manager's convention:
//
//	$ALIBABA_CLOUD_CLI_PLUGINS_DIR, or ~/.aliyun/plugins
//
// Returns "" when no home can be determined; the engine then simply
// omits the user layer.
func userPluginsDir() string {
	if d := os.Getenv(envPluginsDir); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".aliyun", "plugins")
}

// profileHost adapts aliyun-cli's profile/credential resolution to the
// engine's runtime.Host seam. The profile is loaded lazily and once, so
// dry-run (which only reads Region) never triggers credential IO.
type profileHost struct {
	ctx     *cli.Context
	once    sync.Once
	profile config.Profile
	loadErr error
}

func (h *profileHost) load() {
	h.once.Do(func() {
		h.profile, h.loadErr = config.LoadProfileWithContext(h.ctx)
	})
}

// Region returns the profile's default region, or "" if the profile
// cannot be loaded (dry-run tolerates this and resolves region-less).
func (h *profileHost) Region() string {
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
// folded in from ctx after extractHostGlobals.
func (h *profileHost) Settings() runtime.Settings {
	h.load()
	s := runtime.Settings{}
	if h.loadErr == nil {
		s.ReadTimeout = time.Duration(h.profile.ReadTimeout) * time.Second
		s.ConnectTimeout = time.Duration(h.profile.ConnectTimeout) * time.Second
		s.RetryCount = h.profile.RetryCount
		s.EndpointType = h.profile.EndpointType
		s.Language = h.profile.Language
	}
	if h.ctx != nil {
		if f := h.ctx.Flags().Get(config.SkipSecureVerifyName); f != nil && f.IsAssigned() {
			s.SkipSecureVerify = true
		}
		s.UserAgent = buildUserAgent(h.ctx)
	}
	return s
}

// buildUserAgent merges --user-agent with the host AI-mode suffix
// (config + --cli-ai-mode / --no-cli-ai-mode / --cli-ai-user-agent).
func buildUserAgent(ctx *cli.Context) string {
	var parts []string
	if f := ctx.Flags().Get("user-agent"); f != nil {
		if v, ok := f.GetValue(); ok && strings.TrimSpace(v) != "" {
			parts = append(parts, strings.TrimSpace(v))
		}
	}

	forceOn := flagAssigned(ctx, "cli-ai-mode")
	forceOff := flagAssigned(ctx, "no-cli-ai-mode")
	cfg, err := aimode.Load(config.GetConfigDir(ctx))
	if err != nil {
		cfg = aimode.DefaultAiConfig()
	}
	if f := ctx.Flags().Get("cli-ai-user-agent"); f != nil {
		if v, ok := f.GetValue(); ok && strings.TrimSpace(v) != "" {
			if cfg == nil {
				cfg = aimode.DefaultAiConfig()
			}
			cfg.UserAgent = strings.TrimSpace(v)
		}
	}
	if suf := aimode.RequestUserAgentSuffixForCommand(cfg, forceOn, forceOff); suf != "" {
		parts = append(parts, suf)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func flagAssigned(ctx *cli.Context, name string) bool {
	f := ctx.Flags().Get(name)
	return f != nil && f.IsAssigned()
}

// Credential resolves the caller's credential from the profile.
func (h *profileHost) Credential() (credentialsv2.Credential, error) {
	h.load()
	if h.loadErr != nil {
		return nil, h.loadErr
	}
	return h.profile.GetCredential(h.ctx, nil)
}

// sharedEngine is the process-wide engine, built once. The top-level product
// router dispatches through it and resolves only the requested product.
var (
	engineOnce sync.Once
	engineInst *jsoncmd.Engine
)

// Engine returns the shared aliyun-openapi-runtime engine.
func Engine() *jsoncmd.Engine {
	engineOnce.Do(func() {
		engineInst = openapiruntime.NewEngine(openapiruntime.Options{
			BaselineFS:     aliyunopenapimeta.Metadatas,
			BundledBy:      "aliyun-cli " + cli.Version,
			UserPluginsDir: userPluginsDir(),
			OverrideDir:    os.Getenv("ALIYUN_CLI_PLUGINS_DIR_OVERRIDE"),
		}, nil)
	})
	return engineInst
}

// Dispatch runs "<product> <command> [--flags...]" through the engine
// using the profile-backed host. rawArgs is the argv tail starting at
// the product (e.g. os.Args from the product onward).
//
// Host global flags (profile / credential / network) are extracted from
// the tail and applied to ctx first, so the engine only sees its own
// runtime flags and the API parameters.
func Dispatch(ctx *cli.Context, rawArgs []string) error {
	engineArgs := extractHostGlobals(ctx, rawArgs)
	host := &profileHost{ctx: ctx}

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

	return Engine().Dispatch(jsoncmd.Request{
		Args: engineArgs,
		Out:  ctx.Stdout(),
		Lang: lang,
		Host: host,
	})
}

// TryDispatch handles rawArgs via the engine only when the engine can
// resolve the "<product> <command>" pair. It returns handled=false
// (with nil error) otherwise, letting the caller fall back to legacy
// routing (auto-install, built-in openapi, ...).
func TryDispatch(ctx *cli.Context, rawArgs []string) (handled bool, err error) {
	if len(rawArgs) < 2 {
		return false, nil
	}
	if !Engine().Resolvable(rawArgs[0], rawArgs[1]) {
		return false, nil
	}
	return true, Dispatch(ctx, rawArgs)
}

// TryHelp renders the engine's parameter help for "<product> <command>" when the engine can resolve it,
// returning handled=false otherwise so the caller can fall back to the legacy help path.
func TryHelp(ctx *cli.Context, product, command string) (handled bool, err error) {
	if product == "" || command == "" {
		return false, nil
	}
	if !Engine().Resolvable(product, command) {
		return false, nil
	}
	// Forward a requested API version so help renders the selected
	// version's parameters. The user may pass it either as the engine
	// flag --api-version (scanned from the raw argv, mirroring the Go
	// plugin help path) or the host's legacy --version flag.
	args := []string{product, command}
	if v := apiVersionFromArgs(os.Args); v != "" {
		args = append(args, "--api-version", v)
	} else if vf := ctx.Flags().Get("version"); vf != nil {
		if v, ok := vf.GetValue(); ok && v != "" {
			args = append(args, "--api-version", v)
		}
	}
	// The engine's argparser turns --help into a Reserved.Help render.
	args = append(args, "--help")
	return true, Dispatch(ctx, args)
}

// apiVersionFromArgs extracts --api-version from a raw argv, supporting
// both "--api-version X" and "--api-version=X". Returns "" when absent.
func apiVersionFromArgs(argv []string) string {
	const flag = "--api-version"
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == flag {
			if i+1 < len(argv) {
				return argv[i+1]
			}
			return ""
		}
		if strings.HasPrefix(a, flag+"=") {
			return a[len(flag)+1:]
		}
	}
	return ""
}
