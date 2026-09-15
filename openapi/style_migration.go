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
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi/runtimehost"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
)

// engineServedCommands reports the kebab command names the runtime engine
// serves for a product. It is a package-level variable so tests can pin the
// gate deterministically (the real engine metadata is not available in unit
// test environments).
var engineServedCommands = runtimehost.ProductCommands

// styleRenameMaps builds the bidirectional rename tables between the kebab
// option names the engine serves and the legacy raw parameter names of one
// API's top-level parameters. The rename is metadata-driven — a kebab name
// such as "biz-region-id" cannot be derived from "RegionId" by string
// conversion — so both directions come from the same source of truth.
func styleRenameMaps(api *canonicalmeta.API) (kebabToRaw, rawToKebab map[string]string) {
	if api == nil {
		return nil, nil
	}
	kebabToRaw = make(map[string]string)
	rawToKebab = make(map[string]string)
	for i := range api.Parameters {
		p := &api.Parameters[i]
		switch strings.ToLower(p.Location) {
		case "domain", "header":
			continue
		}
		if p.RawName == "" {
			continue
		}
		kebab := kebabOptionName(p)
		if kebab == "" {
			continue
		}
		kebabToRaw[kebab] = p.RawName
		rawToKebab[p.RawName] = kebab
	}
	return kebabToRaw, rawToKebab
}

// kebabOptionName returns the parameter's kebab flag spelling. Options carry
// the authoritative form ("--biz-region-id"); the snake-case Name converted to
// kebab is the fallback for parameters without declared options.
func kebabOptionName(p *canonicalmeta.Parameter) string {
	for _, opt := range p.Options {
		if name := strings.TrimPrefix(opt, "--"); name != "" {
			return name
		}
	}
	return strings.ReplaceAll(p.Name, "_", "-")
}

func keySet(m map[string]string) map[string]bool {
	set := make(map[string]bool, len(m))
	for k := range m {
		set[k] = true
	}
	return set
}

// rebuildStyleEquivalentCommand rewrites the current invocation in the target
// command style: host flags pass through unchanged, API parameter flags are
// renamed through the metadata tables, and values keep their assignments. It
// fails closed — any unmappable API flag yields "" so callers fall back to a
// static example instead of printing a wrong command.
func rebuildStyleEquivalentCommand(product, targetCommand string, ctx *cli.Context, rename map[string]string, validTarget map[string]bool) string {
	if targetCommand == "" || ctx == nil {
		return ""
	}
	parts := []string{"aliyun", strings.ToLower(product), targetCommand}
	if ctx.Flags() != nil {
		for _, f := range ctx.Flags().Flags() {
			if f == nil || !f.IsAssigned() {
				continue
			}
			emitFlagValues(&parts, f.Name, flagAssignedValues(f))
		}
	}
	if ctx.UnknownFlags() != nil {
		for _, f := range ctx.UnknownFlags().Flags() {
			if f == nil || !f.IsAssigned() {
				continue
			}
			name := strings.TrimSuffix(f.Name, "-FILE")
			switch {
			case validTarget[name]:
				// already spelled in the target style
			case rename[name] != "":
				name = rename[name]
			default:
				return ""
			}
			emitFlagValues(&parts, name, flagAssignedValues(f))
		}
	}
	return strings.Join(parts, " ")
}

// flagAssignedValues returns the assigned values of a flag: every value for
// repeatable flags, the single value otherwise, and nil for valueless
// (boolean) flags.
func flagAssignedValues(f *cli.Flag) []string {
	if values := f.GetValues(); len(values) > 0 {
		return values
	}
	if v, ok := f.GetValue(); ok && v != "" {
		return []string{v}
	}
	return nil
}

func emitFlagValues(parts *[]string, name string, values []string) {
	if len(values) == 0 {
		*parts = append(*parts, "--"+name)
		return
	}
	for _, v := range values {
		*parts = append(*parts, "--"+name, quoteShellValue(v))
	}
}

// quoteShellValue keeps shell-safe values bare and quotes the rest, so the
// rebuilt command stays readable and copy-pasteable.
func quoteShellValue(v string) string {
	if safeCommandToken(v) {
		return v
	}
	return shellSingleQuote(v)
}

// styleMixedFlagError adapts the engine's unknown-flag failure when the flag
// is the other command style's parameter name — e.g. the PascalCase wire name
// --RegionId on a kebab command whose metadata-renamed flag is
// --biz-region-id. It carries the style-correct suggestion and, when every
// assigned flag can be mapped, the directly runnable equivalent command.
type styleMixedFlagError struct {
	cause      error
	product    string
	command    string // kebab command name
	flag       string // offending flag without dashes
	suggestion string // kebab option with dashes, e.g. "--biz-region-id"
	equivalent string // equivalent PascalCase command; "" when fail-closed
}

func (e *styleMixedFlagError) Error() string {
	msg := e.cause.Error()
	kebabName := strings.TrimPrefix(e.suggestion, "--")
	if e.equivalent != "" {
		return fmt.Sprintf("%s\n\n--%s is a PascalCase parameter name; kebab commands use --%s. Equivalent command:\n  %s",
			msg, e.flag, kebabName, e.equivalent)
	}
	return fmt.Sprintf("%s\n\n--%s is a PascalCase parameter name; kebab commands use --%s.", msg, e.flag, kebabName)
}

func (e *styleMixedFlagError) Unwrap() error { return e.cause }

func (*styleMixedFlagError) AIRecoveryEligible() {}

func (e *styleMixedFlagError) GetSuggestions() []string {
	if e.suggestion == "" {
		return nil
	}
	return []string{e.suggestion}
}

func (e *styleMixedFlagError) AgentMessage() string {
	return fmt.Sprintf("unknown flag --%s", e.flag)
}

func (e *styleMixedFlagError) AgentSuggestions() []string {
	return e.GetSuggestions()
}

// adaptStyleMixedFlagError converts the engine's unknown-flag failure into a
// style-aware suggestion when the flag is a PascalCase wire name of the same
// API. Every verification step fails closed to the original error: unknown
// product, missing canonical metadata, unmappable flag, or a kebab suggestion
// the engine does not actually serve.
func (c *Commando) adaptStyleMixedFlagError(err error, args []string, ctx *cli.Context) error {
	if err == nil || len(args) < 2 || commandStyle(args[1]) != "kebab" {
		return err
	}
	var profileCase *kebabProfileFlagCaseError
	if errors.As(err, &profileCase) {
		return err
	}
	var unknown *argparser.UnknownFlagError
	if !errors.As(err, &unknown) {
		return err
	}
	if c.library == nil || c.library.canonicalRepo == nil {
		return err
	}
	product, ok := c.library.GetProduct(args[0])
	if !ok {
		return err
	}
	apiName, api := resolveKebabCommandAPI(c.library.canonicalRepo, product.Code, product.Version, args[1])
	if api == nil {
		return err
	}
	kebabToRaw, rawToKebab := styleRenameMaps(api)
	kebabName, ok := rawToKebab[unknown.Flag]
	if !ok || !containsString(unknown.Known, kebabName) {
		return err
	}
	equivalent := rebuildStyleEquivalentCommand(product.Code, apiName, ctx, kebabToRaw, keySet(rawToKebab))
	return &styleMixedFlagError{
		cause:      err,
		product:    strings.ToLower(product.Code),
		command:    args[1],
		flag:       unknown.Flag,
		suggestion: "--" + kebabName,
		equivalent: equivalent,
	}
}

// resolveKebabCommandAPI finds the canonical API behind an engine-served
// kebab command name, matching on the declared cmd_name first and the
// converted API name as fallback.
func resolveKebabCommandAPI(repo canonicalAPIRepository, productCode, version, kebabCommand string) (string, *canonicalmeta.API) {
	index, err := repo.GetVersionIndex(productCode, version)
	if err != nil || index == nil {
		return "", nil
	}
	for apiName, entry := range index.APIs {
		if entry.CmdName == kebabCommand || apiNameToKebab(apiName) == kebabCommand {
			api, err := repo.GetAPI(productCode, version, apiName)
			if err == nil && api != nil {
				return apiName, api
			}
		}
	}
	return "", nil
}

// regionConfusionNote detects the routing-flag mix-up: on kebab commands
// --region only selects the signing/endpoint region, while the API's region
// parameter (e.g. --biz-region-id) must be passed explicitly. It fires only
// when the user did pass --region and a missing required parameter is the
// region-ish one. The PascalCase chain needs no note: there --region feeds
// the wire RegionId through the legacy invoker.
func regionConfusionNote(regionAssigned bool, missingFlags []string) string {
	if !regionAssigned {
		return ""
	}
	for _, flag := range missingFlags {
		if strings.Contains(compactAPITokens(strings.TrimLeft(flag, "-")), "regionid") {
			return fmt.Sprintf("Note: --region only selects the region for signing and endpoint resolution; it does not set the API parameter %s. Pass %s explicitly to set it.", flag, flag)
		}
	}
	return ""
}

// regionConfusionError appends the --region clarification to the engine's
// missing-required message in human output. AI mode reads the note off the
// wrapper and enriches the recovery hint with it.
type regionConfusionError struct {
	cause error
	note  string
}

func (e *regionConfusionError) Error() string { return e.cause.Error() + "\n\n" + e.note }

func (e *regionConfusionError) Unwrap() error { return e.cause }

func (*regionConfusionError) AIRecoveryEligible() {}

// annotateRegionConfusion attaches the --region clarification to the engine's
// missing-required failure when the mix-up pattern matches; otherwise the
// error passes through unchanged. --region assignment is read from the parsed
// flags, which is authoritative, rather than from a raw argv scan.
func annotateRegionConfusion(err error, ctx *cli.Context) error {
	var missing *runtime.MissingRequiredError
	if !errors.As(err, &missing) {
		return err
	}
	regionAssigned := false
	if ctx != nil && ctx.Flags() != nil {
		if f := config.RegionFlag(ctx.Flags()); f != nil && f.IsAssigned() {
			regionAssigned = true
		}
	}
	note := regionConfusionNote(regionAssigned, missing.Flags)
	if note == "" {
		return err
	}
	return &regionConfusionError{cause: err, note: note}
}
