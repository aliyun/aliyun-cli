package openapi

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type machineHelpUtilityDocument struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Kind          string                      `json:"helpLevel"`
	Target        machineHelpTarget           `json:"-"`
	Name          string                      `json:"name"`
	Description   machineHelpLocalizedText    `json:"description"`
	Usage         string                      `json:"usage"`
	Sample        string                      `json:"sample"`
	Query         string                      `json:"query"`
	Commands      []machineHelpCommandSummary `json:"commands"`
	Flags         []machineHelpFlagSummary    `json:"flags"`
	Result        HelpResult                  `json:"result"`
	Next          *HelpNext                   `json:"next"`
	AIModeHint    *machineHelpAIModeHint      `json:"aiModeHint"`
}

func (c *Commando) renderUtilityHelp(ctx *cli.Context, path []string, options cli.HelpOptions, aiMode bool) error {
	document, err := buildUtilityHelpDocument(ctx.Command(), path)
	if err != nil {
		return err
	}
	applyUtilityHelpOptions(document, options)
	target := utilityHelpTarget(path, options)
	setUtilityHelpNext(document, target, aiMode)
	jsonOutput := aiMode || options.Output == cli.HelpOutputJSON
	if !aiMode && jsonOutput {
		hint := cli.NewAIModeHint()
		document.AIModeHint = &machineHelpAIModeHint{Command: hint.Command, Message: hint.Message}
	}
	if jsonOutput {
		return encodeMachineHelpJSON(ctx.Stdout(), document, aiMode)
	}
	if err := renderUtilityHelpText(ctx, document); err != nil {
		return err
	}
	return c.finishCanonicalTextHelp(ctx, aiMode, target)
}

func buildUtilityHelpDocument(root *cli.Command, path []string) (*machineHelpUtilityDocument, error) {
	if root == nil || len(path) == 0 || path[0] != "utils" || len(path) > 2 {
		return nil, fmt.Errorf("invalid utility Help target")
	}
	command := root.GetSubCommand("utils")
	if command == nil {
		return nil, fmt.Errorf("utils command is unavailable")
	}
	if len(path) == 2 {
		command = command.GetSubCommand(path[1])
		if command == nil {
			return nil, fmt.Errorf("%q is not a valid utility command", path[1])
		}
	}
	targetPath := append([]string{"aliyun"}, path...)
	document := &machineHelpUtilityDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "utility",
		Target:        machineHelpTarget{Path: targetPath, RequestedStyle: "utility"},
		Name:          strings.Join(path, " "),
		Description:   localizedText(command.Short.GetData()),
		Usage:         command.Usage,
		Sample:        command.Sample,
	}
	for _, name := range command.SubCommandNames() {
		child := command.GetSubCommand(name)
		if child == nil || child.Hidden {
			continue
		}
		document.Commands = append(document.Commands, machineHelpCommandSummary{
			Path:        append(append([]string(nil), targetPath...), name),
			Name:        name,
			Description: localizedText(child.Short.GetData()),
		})
	}
	for _, flag := range command.Flags().Flags() {
		if flag == nil || flag.Hidden {
			continue
		}
		shorthand := ""
		if flag.Shorthand != 0 {
			shorthand = "-" + string(flag.Shorthand)
		}
		document.Flags = append(document.Flags, machineHelpFlagSummary{
			Name: "--" + flag.Name, Aliases: append([]string(nil), flag.Aliases...), Shorthand: shorthand,
			Category: flag.Category, Visibility: "default", Description: localizedText(flag.Short.GetData()),
		})
	}
	document.Result = HelpResult{Shown: len(document.Commands) + len(document.Flags), Total: len(document.Commands) + len(document.Flags)}
	return document, nil
}

func applyUtilityHelpOptions(document *machineHelpUtilityDocument, options cli.HelpOptions) {
	if document == nil || options.Operation != cli.HelpOperationSearch {
		return
	}
	document.Query = options.SearchQuery
	type entry struct {
		command *machineHelpCommandSummary
		flag    *machineHelpFlagSummary
	}
	candidates := make([]HelpSearchCandidate, 0, len(document.Commands)+len(document.Flags))
	for index := range document.Commands {
		command := document.Commands[index]
		candidates = append(candidates, HelpSearchCandidate{Kind: "command", Name: command.Name, Aliases: command.Aliases,
			DescriptionEN: command.Description.EN, DescriptionZH: command.Description.ZH, Value: entry{command: &command}})
	}
	for index := range document.Flags {
		flag := document.Flags[index]
		aliases := append([]string(nil), flag.Aliases...)
		if flag.Shorthand != "" {
			aliases = append(aliases, flag.Shorthand)
		}
		candidates = append(candidates, HelpSearchCandidate{Kind: "flag", Name: flag.Name,
			Aliases:       aliases,
			DescriptionEN: flag.Description.EN, DescriptionZH: flag.Description.ZH, Value: entry{flag: &flag}})
	}
	projection := ProjectHelpSearchMatches(SearchHelpCandidates(candidates, options.SearchQuery), options.SearchAll)
	document.Commands = nil
	document.Flags = nil
	for _, match := range projection.Matches {
		value := match.Candidate.Value.(entry)
		if value.command != nil {
			document.Commands = append(document.Commands, *value.command)
		} else if value.flag != nil {
			document.Flags = append(document.Flags, *value.flag)
		}
	}
	document.Result = projection.Result
}

func renderUtilityHelpText(ctx *cli.Context, document *machineHelpUtilityDocument) error {
	w := ctx.Stdout()
	if _, err := fmt.Fprintf(w, "%s\n", localizedMachineHelpText(document.Description)); err != nil {
		return err
	}
	if document.Usage != "" {
		if _, err := fmt.Fprintf(w, "\nUsage:\n  aliyun %s\n", document.Usage); err != nil {
			return err
		}
	}
	if document.Query != "" && len(document.Commands) == 0 && len(document.Flags) == 0 {
		if _, err := fmt.Fprintf(w, "\n"+noHelpSearchMatchesFormat+"\n", document.Query); err != nil {
			return err
		}
		return renderHelpProjectionResult(w, "matches", document.Result, document.Next)
	}
	if len(document.Commands) > 0 {
		if _, err := fmt.Fprintln(w, "\nCommands:"); err != nil {
			return err
		}
		for _, command := range document.Commands {
			if _, err := fmt.Fprintf(w, "  %-32s %s\n", command.Name, localizedMachineHelpText(command.Description)); err != nil {
				return err
			}
		}
	}
	if len(document.Flags) > 0 {
		if _, err := fmt.Fprintln(w, "\nFlags:"); err != nil {
			return err
		}
		for _, flag := range document.Flags {
			name := flag.Name
			if flag.Shorthand != "" {
				name += ", " + flag.Shorthand
			}
			if _, err := fmt.Fprintf(w, "  %-32s %s\n", name, localizedMachineHelpText(flag.Description)); err != nil {
				return err
			}
		}
	}
	if document.Sample != "" {
		if _, err := fmt.Fprintf(w, "\nExample:\n  %s\n", document.Sample); err != nil {
			return err
		}
	}
	return renderHelpProjectionResult(w, "matches", document.Result, document.Next)
}
