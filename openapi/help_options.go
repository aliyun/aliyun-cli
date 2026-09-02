package openapi

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

const (
	helpSectionRequest  = string(cli.HelpSectionRequest)
	helpSectionResponse = string(cli.HelpSectionResponse)
)

type helpOptions struct {
	Section         string
	SectionExplicit bool
	Search          string
	// SearchAll removes the search result cap. Every public search enables it;
	// an explicit --help-all remains accepted for compatibility.
	SearchAll bool
	All       bool
	Output    cli.HelpOutput
}

func canonicalHelpOptionAssigned(fs *cli.FlagSet) bool {
	if fs == nil {
		return false
	}
	for _, flag := range []*cli.Flag{
		CliHelpSectionFlag(fs),
		CliHelpSearchFlag(fs),
		CliHelpAllFlag(fs),
	} {
		if flag != nil && flag.IsAssigned() {
			return true
		}
	}
	return false
}

func parseHelpOptions(ctx *cli.Context, target []string) (helpOptions, error) {
	opts := helpOptions{Section: helpSectionRequest, Output: cli.HelpOutputText}
	if ctx == nil || ctx.Flags() == nil {
		return opts, nil
	}

	if flag := CliHelpSectionFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		value, _ := flag.GetValue()
		opts.Section = strings.ToLower(strings.TrimSpace(value))
		opts.SectionExplicit = true
		if opts.Section != helpSectionRequest && opts.Section != helpSectionResponse {
			return helpOptions{}, &cli.HelpOptionError{
				Code:   cli.HelpOptionInvalidSection,
				Option: "--" + CliHelpSectionFlagName,
				Value:  opts.Section,
			}
		}
		if len(target) != 2 {
			return helpOptions{}, fmt.Errorf("--%s requires an API target: `aliyun help <product> <API>` or `aliyun <product> <API> --help`", CliHelpSectionFlagName)
		}
	}

	if flag := CliHelpSearchFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		value, _ := flag.GetValue()
		opts.Search = strings.TrimSpace(value)
		if opts.Search == "" {
			return helpOptions{}, &cli.HelpOptionError{
				Code:   cli.HelpOptionEmptySearch,
				Option: "--" + CliHelpSearchFlagName,
			}
		}
	}

	if flag := CliHelpAllFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		if opts.Search != "" {
			// Search + All composes: the keyword keeps filtering while the
			// result cap is removed.
			opts.SearchAll = true
		} else {
			opts.All = true
		}
	}
	if opts.Search != "" {
		opts.SearchAll = true
	}

	if flag := CliOutputFlag(ctx.Flags()); flag != nil && flag.IsAssigned() {
		value, _ := flag.GetValue()
		value = strings.TrimSpace(value)
		if value != string(cli.HelpOutputJSON) {
			return helpOptions{}, &cli.HelpOptionError{
				Code:   cli.HelpOptionInvalidOutput,
				Option: "--" + CliOutputFlagName,
				Value:  value,
			}
		}
		opts.Output = cli.HelpOutputJSON
	}

	return opts, nil
}
