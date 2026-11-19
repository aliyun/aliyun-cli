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
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

// cliInputJSONPair is one parameter read from a --cli-input-json file. Order is
// preserved so argv expansion and error reporting stay deterministic.
type cliInputJSONPair struct {
	Name  string
	Value string
}

// cliInputJSONError reports the flag name, file path and offending parameter
// without ever echoing the file contents, which may carry secrets.
type cliInputJSONError struct {
	Path   string
	Key    string
	Reason string
	Err    error
}

func (e *cliInputJSONError) Error() string {
	var b strings.Builder
	b.WriteString("--" + CliInputJSONFlagName + ": ")
	if e.Path != "" {
		b.WriteString(e.Path + ": ")
	}
	if e.Key != "" {
		fmt.Fprintf(&b, "parameter %q: ", e.Key)
	}
	b.WriteString(e.Reason)
	if e.Err != nil {
		b.WriteString(": " + e.Err.Error())
	}
	return b.String()
}

func (e *cliInputJSONError) Unwrap() error { return e.Err }

// cliInputJSONKeyPattern keeps expanded argv tokens well formed: a key must
// start with a letter or digit, so a leading '-' cannot become "---x" and a
// key cannot smuggle in '=' or whitespace. Nested names such as Tag.1.Key,
// kebab-case names, and plugin flags keep working.
var cliInputJSONKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// readCliInputJSONPairs parses and validates the assigned --cli-input-json file.
// legacy selects the PascalCase route, where RegionId is also a root flag.
// It returns nil pairs when the flag is not assigned.
func readCliInputJSONPairs(ctx *cli.Context, legacy bool) ([]cliInputJSONPair, error) {
	flag := CliInputJSONFlag(ctx.Flags())
	if flag == nil || !flag.IsAssigned() {
		return nil, nil
	}
	path, ok := flag.GetValue()
	path = strings.TrimSpace(path)
	if !ok || path == "" {
		return nil, &cliInputJSONError{Reason: "file path must not be empty"}
	}
	pairs, err := parseCliInputJSONFile(path)
	if err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		if err := validateCliInputJSONKey(ctx, pair.Name, path, legacy); err != nil {
			return nil, err
		}
	}
	return pairs, nil
}

// parseCliInputJSONFile reads a flat JSON object of parameter name to scalar
// value. Numbers keep their original literal so large integers do not lose
// precision on their way to the request.
func parseCliInputJSONFile(path string) ([]cliInputJSONPair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &cliInputJSONError{Path: path, Reason: "cannot read parameters file", Err: err}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return nil, &cliInputJSONError{Path: path, Reason: "invalid JSON", Err: err}
	}
	if delim, isDelim := tok.(json.Delim); !isDelim || delim != '{' {
		return nil, &cliInputJSONError{Path: path, Reason: "root must be a JSON object mapping parameter names to values"}
	}

	pairs := make([]cliInputJSONPair, 0, 8)
	positions := make(map[string]int, 8)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, &cliInputJSONError{Path: path, Reason: "invalid JSON", Err: err}
		}
		key, _ := keyTok.(string)
		if key == "" {
			return nil, &cliInputJSONError{Path: path, Reason: "parameter names must not be empty"}
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, &cliInputJSONError{Path: path, Key: key, Reason: "invalid JSON", Err: err}
		}
		value, err := cliInputJSONScalar(path, key, raw)
		if err != nil {
			return nil, err
		}
		// Standard JSON decoding keeps the last value for duplicate keys.
		if at, seen := positions[key]; seen {
			pairs[at].Value = value
			continue
		}
		positions[key] = len(pairs)
		pairs = append(pairs, cliInputJSONPair{Name: key, Value: value})
	}
	if _, err := dec.Token(); err != nil {
		return nil, &cliInputJSONError{Path: path, Reason: "invalid JSON", Err: err}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, &cliInputJSONError{Path: path, Reason: "unexpected trailing content after the JSON object"}
	}
	return pairs, nil
}

func cliInputJSONScalar(path, key string, raw json.RawMessage) (string, error) {
	text := strings.TrimSpace(string(raw))
	switch {
	case text == "null":
		return "", &cliInputJSONError{Path: path, Key: key, Reason: "null values are not supported; pass this parameter on the command line instead"}
	case strings.HasPrefix(text, "["), strings.HasPrefix(text, "{"):
		return "", &cliInputJSONError{Path: path, Key: key, Reason: "only string, number, and boolean values are supported; pass this parameter on the command line instead (RESTful HTTP bodies can use --body-file)"}
	case strings.HasPrefix(text, "\""):
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", &cliInputJSONError{Path: path, Key: key, Reason: "invalid string value", Err: err}
		}
		return value, nil
	case text == "true" || text == "false":
		return text, nil
	default:
		// Everything else received a syntactically valid JSON value, so it is
		// a number literal; keep the exact spelling.
		return text, nil
	}
}

func validateCliInputJSONKey(ctx *cli.Context, name, path string, legacy bool) error {
	if !cliInputJSONKeyPattern.MatchString(name) {
		return &cliInputJSONError{Path: path, Key: name, Reason: "parameter names must start with a letter or digit and may only contain letters, digits, '-', '_', and '.'"}
	}
	if strings.HasSuffix(name, "-FILE") {
		return &cliInputJSONError{Path: path, Key: name, Reason: "names ending in -FILE are reserved for single-parameter file input; pass this parameter on the command line instead"}
	}
	// RegionId is both the standard API parameter and a root flag that selects
	// the request region. The legacy route applies it like a command line
	// --RegionId, so the original issue example keeps working. Kebab-case and
	// plugin commands spell the parameter differently (for example
	// --biz-region-id), so the key is rejected there with a pointer to the file.
	if name == config.RegionIdFlagName {
		if legacy {
			return nil
		}
		return &cliInputJSONError{Path: path, Key: name, Reason: "RegionId is only supported by legacy PascalCase commands; for kebab-case commands use the API's own region parameter in kebab-case form (for example biz-region-id), or pass --region on the command line for the endpoint/signing region"}
	}
	// The host flag set below only covers visible globals, so hidden engine flags
	// such as all-pages would otherwise be expanded into argv and hijack the
	// option. The legacy PascalCase route registers parameters through the V1 SDK
	// and keeps its own naming rules.
	if !legacy && argparser.IsReservedFlag(name) {
		return &cliInputJSONError{Path: path, Key: name, Reason: fmt.Sprintf("conflicts with the global flag --%s; pass this parameter on the command line instead", name)}
	}
	if ctx != nil && ctx.Flags() != nil && ctx.Flags().Get(name) != nil {
		return &cliInputJSONError{Path: path, Key: name, Reason: fmt.Sprintf("conflicts with the global flag --%s; pass this parameter on the command line instead", name)}
	}
	return nil
}

// injectCliInputJSONParams merges file values into the legacy PascalCase path as
// dynamic parameter flags. Explicit command line parameters win; file values only
// fill parameters the user did not assign.
func injectCliInputJSONParams(ctx *cli.Context) error {
	pairs, err := readCliInputJSONPairs(ctx, true)
	if err != nil || len(pairs) == 0 {
		return err
	}
	unknown := ctx.UnknownFlags()
	if unknown == nil {
		return &cliInputJSONError{Reason: "parameter flags are unavailable for this command"}
	}
	for _, pair := range pairs {
		// A command line --RegionId-FILE registers "RegionId" as an alias in the
		// unknown flag set. Writing the root flag then would override that
		// explicit value and make CheckApiParamWithBuildInArgs add a duplicate
		// flag, which panics. Such invocations keep the generic parameter route.
		if pair.Name == config.RegionIdFlagName && unknown.Get(pair.Name) == nil {
			if root := config.RegionIdFlag(ctx.Flags()); root != nil {
				if !root.IsAssigned() {
					root.SetValue(pair.Value)
					root.SetAssigned(true)
				}
				continue
			}
		}
		flag := unknown.Get(pair.Name)
		if flag == nil {
			flag, err = unknown.AddByName(pair.Name)
			if err != nil {
				return &cliInputJSONError{Key: pair.Name, Reason: "cannot register parameter flag", Err: err}
			}
		} else if flag.IsAssigned() {
			continue
		}
		flag.SetValue(pair.Value)
		flag.SetAssigned(true)
	}
	return nil
}

// expandCliInputJSONArgv rewrites the raw argv tail used by the kebab engine and
// installed-plugin routes: the flag is replaced in place by ordinary parameter
// flags, so a parser that only understands parameters never sees the host flag.
// The host flag is stripped even when the file contributes no parameters.
// Parameters already present in argv keep their explicit command line value.
func expandCliInputJSONArgv(ctx *cli.Context, argv []string) ([]string, error) {
	flag := CliInputJSONFlag(ctx.Flags())
	if flag == nil || !flag.IsAssigned() {
		return argv, nil
	}
	pairs, err := readCliInputJSONPairs(ctx, false)
	if err != nil {
		return argv, err
	}
	explicit := make(map[string]bool, len(argv))
	for _, arg := range argv {
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		// The parser accepts "=" and ":" as value separators.
		name := strings.TrimPrefix(arg, "--")
		if cut := strings.IndexAny(name, "=:"); cut >= 0 {
			name = name[:cut]
		}
		explicit[name] = true
	}
	expanded := make([]string, 0, len(pairs)*2)
	for _, pair := range pairs {
		if explicit[pair.Name] {
			continue
		}
		expanded = append(expanded, "--"+pair.Name, pair.Value)
	}

	out := make([]string, 0, len(argv)+len(expanded))
	inserted := false
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			if !inserted {
				out = append(out, expanded...)
			}
			out = append(out, argv[i:]...)
			return out, nil
		}
		if arg == "--"+CliInputJSONFlagName {
			if !inserted {
				out = append(out, expanded...)
				inserted = true
			}
			// The root parser consumed the next token as this flag's value.
			if i+1 < len(argv) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--"+CliInputJSONFlagName+"=") || strings.HasPrefix(arg, "--"+CliInputJSONFlagName+":") {
			if !inserted {
				out = append(out, expanded...)
				inserted = true
			}
			continue
		}
		out = append(out, arg)
	}
	if !inserted {
		out = append(out, expanded...)
	}
	return out, nil
}

// rejectCliInputJSONLocalRoute refuses the flag on root subcommands (configure,
// version, extensions, ...) where product API parameters cannot apply. The
// generic parser cannot make this decision, so it runs on the original argv
// before parsing and leaves Help requests untouched.
func rejectCliInputJSONLocalRoute(ctx *cli.Context, args []string) error {
	if ctx == nil || ctx.Flags() == nil || CliInputJSONFlag(ctx.Flags()) == nil {
		return nil
	}
	if !cliInputJSONFlagPresent(args, ctx.Flags()) {
		return nil
	}
	// The "help" command is a local command even though it also asks for Help;
	// A1 requires the flag to fail there instead of being ignored. Help options
	// such as --help keep precedence below.
	first := cliInputJSONFirstPositional(args, ctx.Flags())
	if first == "help" {
		return cli.NewErrorWithTip(
			&cliInputJSONError{Reason: "only applies to product API commands"},
			"example: aliyun ecs CreateDeploymentSet --cli-input-json create-deployment-set.json")
	}
	if rawHelpRequested(args) {
		return nil
	}
	if first == "" {
		return nil
	}
	if ctx.Command() != nil && ctx.Command().GetSubCommand(first) != nil {
		return cli.NewErrorWithTip(
			&cliInputJSONError{Reason: fmt.Sprintf("only applies to product API commands, not %q", first)},
			"example: aliyun ecs CreateDeploymentSet --cli-input-json create-deployment-set.json")
	}
	return nil
}

// rejectCliInputJSONVersionRoute refuses the flag on the plugin/engine "version"
// route. That route never runs an API, so the file could not contribute
// parameters, and expanding it would inject values into the version invocation
// (or fail on a file the version path does not need).
func rejectCliInputJSONVersionRoute(ctx *cli.Context, apiOrMethod string) error {
	if apiOrMethod != "version" || ctx == nil || ctx.Flags() == nil {
		return nil
	}
	flag := CliInputJSONFlag(ctx.Flags())
	if flag == nil || !flag.IsAssigned() {
		return nil
	}
	return cli.NewErrorWithTip(
		&cliInputJSONError{Reason: `only applies to product API commands, not "version"`},
		"example: aliyun ecs CreateDeploymentSet --cli-input-json create-deployment-set.json")
}

// cliInputJSONFlagTakesValue mirrors the parser arity rule: a registered flag is
// unary unless its AssignedMode is AssignedNone, and an unrecognized flag is
// treated as unary.
func cliInputJSONFlagTakesValue(flags *cli.FlagSet, arg string) bool {
	trimmed := strings.TrimLeft(arg, "-")
	name := trimmed
	if cut := strings.IndexAny(trimmed, "=:"); cut >= 0 {
		// The value is attached to the option token itself.
		return false
	}
	if flags == nil {
		return true
	}
	if !strings.HasPrefix(arg, "--") && len([]rune(name)) == 1 {
		flag := flags.GetByShorthand([]rune(name)[0])
		if flag == nil {
			return true
		}
		return flag.AssignedMode != cli.AssignedNone
	}
	if name == "" {
		return false
	}
	flag := flags.Get(name)
	if flag == nil {
		return true
	}
	return flag.AssignedMode != cli.AssignedNone
}

// cliInputJSONConsumesNext mirrors cli/parser.go readNext: an option token takes
// the following token as its value only when it needs one and that token is not
// itself an option. A "--xxx" token is always parsed as a flag, and a
// dash-leading value candidate (single dash, no '=' or ':') is only consumed by
// a known long flag with a value.
func cliInputJSONConsumesNext(flags *cli.FlagSet, arg, next string) bool {
	if !cliInputJSONFlagTakesValue(flags, arg) {
		return false
	}
	if strings.HasPrefix(next, "--") {
		return false
	}
	if strings.HasPrefix(next, "-") && len(next) > 1 && !strings.ContainsAny(next, "=:") {
		if flags == nil || !strings.HasPrefix(arg, "--") {
			return false
		}
		return flags.Get(strings.TrimPrefix(arg, "--")) != nil
	}
	return next != "-"
}

func cliInputJSONFlagPresent(args []string, flags *cli.FlagSet) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return false
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if cut := strings.IndexAny(name, "=:"); cut >= 0 {
			if name[:cut] == CliInputJSONFlagName {
				return true
			}
			continue
		}
		if name == CliInputJSONFlagName {
			return true
		}
		if i+1 < len(args) && cliInputJSONConsumesNext(flags, arg, args[i+1]) {
			i++
		}
	}
	return false
}

func cliInputJSONFirstPositional(args []string, flags *cli.FlagSet) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return ""
		}
		if strings.HasPrefix(arg, "-") {
			if i+1 < len(args) && cliInputJSONConsumesNext(flags, arg, args[i+1]) {
				i++
			}
			continue
		}
		return arg
	}
	return ""
}
