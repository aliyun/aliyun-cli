package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
)

// Guard extensions at the root, before their own parsers, nested commands,
// credential resolution, auto-installation or subprocesses can run. These
// commands never enter the OpenAPI/installed-plugin execution path.
func attachExtensionSafetyPolicy(root *cli.Command, extensions map[string]bool) {
	previous := root.BeforeParseRoute
	root.BeforeParseRoute = func(ctx *cli.Context, args []string) (bool, error) {
		if err := checkExtensionSafetyPolicy(ctx, args, extensions); err != nil {
			return true, err
		}
		if previous != nil {
			return previous(ctx, args)
		}
		return false, nil
	}
}

func checkExtensionSafetyPolicy(ctx *cli.Context, args []string, extensions map[string]bool) error {
	if ctx.Completion() != nil {
		return nil
	}
	// Resolve the root command with the same parser used for dispatch, on
	// copied flags so this preflight cannot change the provider's parser state.
	probe := cli.NewCommandContext(io.Discard, io.Discard)
	probe.EnterCommand(&cli.Command{EnableUnknownFlag: true})
	probe.SetInvocationArgs(args)
	for _, flag := range ctx.Flags().Flags() {
		if flag.Name == "help" {
			continue
		}
		copy := *flag
		copy.SetValues(append([]string(nil), flag.GetValues()...))
		probe.Flags().Add(&copy)
	}
	parser := cli.NewParser(args, probe)
	parser.SetAllowUnknown(true)
	product, _, err := parser.ReadNextArg()
	if err != nil || !extensions[product] {
		return nil // Leave unrelated commands and root syntax errors to dispatch.
	}
	args = parser.GetRemains()
	configDir := config.GetConfigDir(probe)
	skipConfirm := probe.Flags().Get("yes").IsAssigned() ||
		os.Getenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM") == "1" ||
		strings.EqualFold(os.Getenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM"), "true")
	var command, providerArgs []string
	literal := false
	for i := 0; i < len(args); i++ {
		token := args[i]
		if literal {
			command = append(command, token)
			providerArgs = append(providerArgs, token)
			continue
		}
		if token == "--" {
			literal = true
			providerArgs = append(providerArgs, token)
			continue
		}
		name, value, inline := cli.SplitStringWithPrefix(token, "=:")
		var flag *cli.Flag
		if strings.HasPrefix(name, "--") {
			flag = ctx.Flags().Get(strings.TrimPrefix(name, "--"))
		} else if len(name) == 2 && name[0] == '-' {
			flag = ctx.Flags().GetByShorthand(rune(name[1]))
		}
		// Extension options belong to the provider. Only host configuration
		// and approval options have host semantics here (notably --version
		// is a boolean in many extensions, but a value option in OpenAPI).
		if flag != nil && flag.Category != "config" && flag.Name != "yes" {
			flag = nil
		}
		if flag != nil {
			if flag.AssignedMode != cli.AssignedNone && !inline {
				if i+1 >= len(args) {
					return fmt.Errorf("missing value for %s", name)
				}
				i++
				value = args[i]
			}
			switch flag.Name {
			case config.ConfigurePathFlagName:
				configDir = filepath.Dir(value)
			case "yes":
				skipConfirm = true
			}
			continue
		}
		providerArgs = append(providerArgs, token)
		// Unknown provider flags have no host schema. Do not guess whether the
		// following token is a value or a subcommand; retain positional tokens.
		if !strings.HasPrefix(token, "-") {
			command = append(command, token)
		}
	}
	// Only unambiguous, standalone introspection is exempt. In particular,
	// an arbitrary provider option whose value is "--help" must not bypass deny.
	if len(providerArgs) == 0 {
		return nil
	}
	if len(providerArgs) == 1 {
		switch providerArgs[0] {
		case "help", "--help", "-h", "version", "--version":
			return nil
		}
	}
	policy, err := safety.LoadEffectivePolicy(configDir)
	if err != nil {
		return fmt.Errorf("load safety policy failed: %w", err)
	}
	return safety.CheckAndConfirm(ctx, policy, safety.CommandInfo{
		Product: product, ApiOrMethod: strings.Join(command, ":"),
	}, skipConfirm)
}
