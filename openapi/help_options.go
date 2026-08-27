package openapi

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

const (
	helpSectionRequest  = "request"
	helpSectionResponse = "response"
)

type helpOptions struct {
	Section         string
	SectionExplicit bool
	Search          string
	All             bool
}

func parseHelpOptions(ctx *cli.Context, target []string) (helpOptions, error) {
	opts := helpOptions{Section: helpSectionRequest}
	if ctx == nil || ctx.Flags() == nil {
		return opts, nil
	}

	if flag := CliHelpSectionFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		value, _ := flag.GetValue()
		opts.Section = strings.ToLower(strings.TrimSpace(value))
		opts.SectionExplicit = true
		if opts.Section != helpSectionRequest && opts.Section != helpSectionResponse {
			return helpOptions{}, fmt.Errorf("--%s must be request or response", CliHelpSectionFlagName)
		}
		if len(target) != 2 {
			return helpOptions{}, fmt.Errorf("--%s requires a product and an API", CliHelpSectionFlagName)
		}
	}

	if flag := CliHelpSearchFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		value, _ := flag.GetValue()
		opts.Search = strings.TrimSpace(value)
		if opts.Search == "" {
			return helpOptions{}, fmt.Errorf("--%s requires a non-empty keyword", CliHelpSearchFlagName)
		}
	}

	if flag := CliHelpAllFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		opts.All = true
		if len(target) == 2 {
			return helpOptions{}, fmt.Errorf("--%s is only valid for root or product Help", CliHelpAllFlagName)
		}
	}

	return opts, nil
}
