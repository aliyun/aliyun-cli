package lib

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

func ossUnknownOptionError(key, command string) error {
	helpCommand := "aliyun oss"
	if command != "" {
		helpCommand += " " + command
	}
	help := fmt.Sprintf("Run '%s --help' for supported oss options.", helpCommand)
	known := ossKnownOptionNames()
	if _, ok := known[key]; ok {
		return fmt.Errorf("invalid option %s for oss commands.\n%s", key, help)
	}

	// This set is for diagnostics only; unsupported OpenAPI flags must not be stripped.
	hostFlags := cli.NewFlagSet()
	config.AddFlags(hostFlags)
	openapi.AddFlags(hostFlags)
	for _, flag := range hostFlags.Flags() {
		for _, name := range flag.GetFormations() {
			if key == name {
				return fmt.Errorf("%s is not an oss command option.\n%s", key, help)
			}
		}
	}

	candidates := make([]string, 0, len(known))
	for name := range known {
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)
	suggester := cli.NewSuggester(key, cli.DefaultSuggestDistance)
	for _, name := range candidates {
		suggester.UnifyApply(name)
	}
	suggestions := suggester.GetResults()
	if len(suggestions) == 0 {
		suggestions, _ = cli.PrefixSuggestions(key, candidates, cli.DefaultSuggestLimit)
	}
	if len(suggestions) > cli.DefaultSuggestLimit {
		suggestions = suggestions[:cli.DefaultSuggestLimit]
	}
	if len(suggestions) > 0 {
		return fmt.Errorf("invalid option %s for oss commands, did you mean %s?", key, strings.Join(suggestions, " or "))
	}
	return fmt.Errorf("invalid option %s for oss commands.\n%s", key, help)
}
