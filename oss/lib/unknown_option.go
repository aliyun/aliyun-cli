package lib

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/openapi"
)

func ossUnknownOptionError(key, command string) error {
	commandOptions := ossCommandOptionNames(command)
	helpCommand := "aliyun oss"
	scope := "oss commands"
	if commandOptions != nil {
		helpCommand += " " + command
		scope = "oss " + command
	}
	help := fmt.Sprintf("Run '%s --help' for supported oss options.", helpCommand)
	invalid := fmt.Sprintf("invalid option %s for %s", key, scope)
	known := ossKnownOptionNames()
	if _, ok := known[key]; ok {
		return fmt.Errorf("%s.\n%s", invalid, help)
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

	candidates := make([]string, 0, len(commandOptions))
	for name := range commandOptions {
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
		return fmt.Errorf("%s, did you mean %s?", invalid, strings.Join(suggestions, " or "))
	}
	return fmt.Errorf("%s.\n%s", invalid, help)
}

func ossCommandOptionNames(command string) map[string]struct{} {
	manager := CommandManager{}
	manager.Init()
	cmd, ok := manager.commandMap[command]
	if !ok {
		return nil
	}
	valid := reflect.ValueOf(cmd).Elem().FieldByName("command").FieldByName("validOptionNames")
	names := make(map[string]struct{}, valid.Len()*2)
	for i := 0; i < valid.Len(); i++ {
		option, ok := OptionMap[valid.Index(i).String()]
		if !ok {
			continue
		}
		for _, name := range []string{option.nameAlias, option.name} {
			if name != "" {
				names[name] = struct{}{}
			}
		}
	}
	return names
}
