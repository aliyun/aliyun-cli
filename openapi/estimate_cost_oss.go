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
	"fmt"
	"regexp"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// OSS is not a POP product: its real API is S3-style RESTful with its own
// signature, `aliyun oss` is a built-in ossutil bridge that shadows the
// generic OpenAPI product routing, and the embedded metadata carries no OSS
// api definitions. None of that blocks cost estimation though — a quote only
// needs the api triple + parameters sent to CloudControl GetApiPrice, never
// an actual OSS call. So the bridge command gets a fallthrough Run handler:
// when the first arg is an OpenAPI-style PascalCase name and --estimate-cost
// is assigned, the call is routed to the quote pipeline; everything else
// keeps the legacy bridge behavior (subcommands like `mb`/`cp` are matched
// by the framework before this handler and are never affected).
const (
	ossPopCode           = "Oss"
	ossDefaultPopVersion = "2019-05-17"
)

// PascalCase OpenAPI action name, e.g. CreateBucket. ossutil subcommands are
// all lowercase short words, so this cannot collide with them.
var ossApiNameRegexp = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// AttachOssEstimateCost wires --estimate-cost support onto the built-in
// `oss` bridge command. Estimate-cost/output flags are registered on the
// command's own flag set (they are not persistent on root, so they would not
// be inherited), and unknown flags are enabled so OpenAPI parameters like
// --StorageClass can be collected without being declared in metadata.
func (c *Commando) AttachOssEstimateCost(cmd *cli.Command) {
	cmd.EnableUnknownFlag = true
	fs := cmd.Flags()
	fs.Add(NewEstimateCostFlag())
	fs.Add(NewEstimateCostContextFlag())
	fs.Add(NewQuietFlag())
	fs.Add(NewQueryFlag())
	fs.Add(NewOutputFlag())
	cmd.Run = func(ctx *cli.Context, args []string) error {
		return c.processOssBridgeFallthrough(ctx, cmd, args)
	}
}

// processOssBridgeFallthrough runs when no ossutil subcommand matched.
// Without --estimate-cost it reproduces the legacy behavior exactly:
// unknown token -> invalid command error, nothing -> command help.
func (c *Commando) processOssBridgeFallthrough(ctx *cli.Context, cmd *cli.Command, args []string) error {
	if !EstimateCostFlag(ctx.Flags()).IsAssigned() {
		if len(args) > 0 {
			return cli.NewInvalidCommandError(args[0], ctx)
		}
		cmd.PrintHead(ctx)
		cmd.PrintUsage(ctx)
		cmd.PrintSubCommands(ctx)
		cmd.PrintFlags(ctx)
		cmd.PrintTail(ctx)
		return nil
	}
	if len(args) == 0 || !ossApiNameRegexp.MatchString(args[0]) {
		got := "<none>"
		if len(args) > 0 {
			got = args[0]
		}
		return cli.NewErrorWithTip(
			fmt.Errorf("--estimate-cost on `aliyun oss` requires an OpenAPI name, got `%s`", got),
			"use `aliyun oss <ApiName> --Param value ... --estimate-cost`, e.g. `aliyun oss CreateBucket --StorageClass Standard --estimate-cost`; ossutil file commands (mb/cp/...) do not support cost estimation")
	}
	return c.processOssEstimateCost(ctx, args[0])
}

func (c *Commando) processOssEstimateCost(ctx *cli.Context, apiName string) error {
	// Honor --profile etc. typed after `oss` — profile/mode are persistent
	// root flags, so they were parsed into this context by the framework.
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return cli.NewErrorWithTip(err,
			"cost estimation needs a configured profile; run `aliyun configure` first")
	}

	parameters := buildOssEstimateCostParameters(ctx, &profile)

	pricingContext, err := buildPricingContext(ctx)
	if err != nil {
		return err
	}
	if len(pricingContext) > 0 {
		parameters["PricingContext"] = pricingContext
	}

	// The pricing registry keys the triple on the metadata product identity
	// (`Oss` + default version). The embedded metadata has the product entry
	// even though it has no per-api definitions; fall back to constants in
	// case a stripped metadata build drops the entry.
	popCode := ossPopCode
	popVersion := ossDefaultPopVersion
	if product, found := c.library.GetProduct("oss"); found {
		if product.Code != "" {
			popCode = product.Code
		}
		if product.Version != "" {
			popVersion = product.Version
		}
	}

	out, err := invokeEstimateCost(ctx, &profile, popCode, popVersion, apiName, parameters)
	if err != nil {
		return err
	}
	if err := printEstimateCostResult(ctx, out); err != nil {
		return err
	}
	return estimateCostBusinessError(out)
}

// buildOssEstimateCostParameters collects OpenAPI parameters from the
// unknown flags (there is no OSS api metadata to type them against).
// `--region` is the bridge's native region selector, so it doubles as the
// RegionId fallback instead of being forwarded as a bogus parameter.
func buildOssEstimateCostParameters(ctx *cli.Context, profile *config.Profile) map[string]interface{} {
	parameters := make(map[string]interface{})
	regionOverride := ""
	if ctx.UnknownFlags() != nil {
		for _, f := range ctx.UnknownFlags().Flags() {
			if !f.IsAssigned() {
				continue
			}
			if f.Name == "region" {
				if v, ok := f.GetValue(); ok {
					regionOverride = v
				}
				continue
			}
			if values := f.GetValues(); len(values) > 1 {
				parameters[f.Name] = values
				continue
			}
			if v, ok := f.GetValue(); ok && v != "" {
				parameters[f.Name] = v
			}
		}
	}
	if _, ok := parameters["RegionId"]; !ok {
		if regionOverride != "" {
			parameters["RegionId"] = regionOverride
		} else if profile.RegionId != "" {
			parameters["RegionId"] = profile.RegionId
		}
	}
	return parameters
}
