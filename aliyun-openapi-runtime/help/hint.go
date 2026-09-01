package help

import (
	"fmt"
	"io"
	"strings"
)

const (
	runtimeAIModeEnableCommand  = "export ALIBABA_CLOUD_CLI_AI_MODE=1"
	runtimeAIModeEnableMessage  = "Enable AI Mode for compact Help, structured JSON errors, and actionable recovery guidance."
	runtimeAIModeEnableTextHint = "For AI agents, run:\n  " + runtimeAIModeEnableCommand +
		"\n\nThis enables compact Help, structured JSON errors, and actionable recovery guidance."
)

type helpHintKind uint8

const (
	helpHintProduct helpHintKind = iota + 1
	helpHintAPI
	helpHintParameter
)

type helpHintTarget struct {
	kind             helpHintKind
	target           Target
	parameter        string
	requestedVersion string
	explicitSection  bool
	section          Section
}

type helpHintChild struct {
	kind      helpHintKind
	action    string
	parameter string
}

func buildHelpNext(target helpHintTarget, options HelpOptions, result Result, defaultEligible bool, child *helpHintChild) *Next {
	options = options.normalized()
	if target.explicitSection && options.All && options.Search == "" {
		return nil
	}
	if options.Search == "" && !options.All && !defaultEligible {
		return nil
	}

	jsonOutput := options.Format == FormatJSON && !options.AIMode
	next := &Next{operation: helpHintOperation(options)}
	next.Search = buildHelpHintCommand(target, "search", "<keyword>", false, jsonOutput)
	switch {
	case options.Search != "":
		if !options.All {
			next.SearchAll = buildHelpHintCommand(target, "search", options.Search, true, jsonOutput)
		}
	case options.All:
		// All intentionally offers only Search at the same level.
	default:
		next.ShowAll = buildHelpHintCommand(target, "all", "", false, jsonOutput)
	}

	if options.Search != "" && child != nil {
		childTarget := target
		childTarget.explicitSection = false
		childTarget.section = SectionRequest
		switch child.kind {
		case helpHintAPI:
			childTarget.kind = helpHintAPI
			childTarget.target.API = child.action
			childTarget.parameter = ""
		case helpHintParameter:
			childTarget.kind = helpHintParameter
			childTarget.parameter = child.parameter
		default:
			childTarget = helpHintTarget{}
		}
		if childTarget.kind != 0 {
			next.ChildSearch = buildHelpHintCommand(childTarget, "search", "<keyword>", false, jsonOutput)
			next.childKind = child.kind
			if child.kind == helpHintAPI {
				next.childName = child.action
			} else {
				next.childName = strings.TrimLeft(child.parameter, "-")
			}
		}
	}
	return next
}

func helpHintOperation(options HelpOptions) string {
	if options.Search != "" {
		return "search"
	}
	if options.All {
		return "all"
	}
	return "default"
}

func buildHelpHintCommand(target helpHintTarget, operation, query string, searchAll, jsonOutput bool) string {
	args := []string{"aliyun"}
	if target.explicitSection {
		args = append(args, "help", target.target.Product, target.target.API)
	} else {
		switch target.kind {
		case helpHintProduct:
			args = append(args, target.target.Product)
		case helpHintAPI, helpHintParameter:
			args = append(args, target.target.Product, target.target.API)
		}
	}
	if target.requestedVersion != "" {
		args = append(args, "--api-version", shellToken(target.requestedVersion))
	}
	if target.kind == helpHintParameter {
		args = append(args, "--"+strings.TrimLeft(target.parameter, "-"))
	}
	if target.explicitSection {
		args = append(args, "--cli-section", string(target.section))
	}
	switch operation {
	case "default":
		if !target.explicitSection {
			args = append(args, "--help")
		}
	case "all":
		args = append(args, "--help-all")
	case "search":
		queryToken := shellToken(query)
		if query == "<keyword>" {
			queryToken = query
		}
		args = append(args, "--help-search", queryToken)
		if searchAll {
			args = append(args, "--help-all")
		}
	}
	if jsonOutput {
		args = append(args, "--cli-output", "json")
	}
	return strings.Join(args, " ")
}

func helpHintTargetForProduct(document *ProductDocument, options HelpOptions) helpHintTarget {
	return helpHintTarget{
		kind: helpHintProduct, target: document.Target, requestedVersion: options.RequestedVersion,
	}
}

func helpHintTargetForAPI(target Target, options HelpOptions) helpHintTarget {
	return helpHintTarget{
		kind: helpHintAPI, target: target, requestedVersion: options.RequestedVersion,
		explicitSection: options.ExplicitSection, section: options.Section,
	}
}

func helpHintTargetForParameter(target Target, parameter string, options HelpOptions) helpHintTarget {
	return helpHintTarget{
		kind: helpHintParameter, target: target, parameter: strings.TrimLeft(parameter, "-"),
		requestedVersion: options.RequestedVersion,
	}
}

func uniqueExactAPIChild(apis []APISummary, query string) *helpHintChild {
	search := searchText(query)
	count := 0
	command := ""
	for _, api := range apis {
		if bestSearchRank(search, api.Name, api.Command) != 1 {
			continue
		}
		count++
		command = api.Command
	}
	if count != 1 || command == "" {
		return nil
	}
	return &helpHintChild{kind: helpHintAPI, action: command}
}

func uniqueExactParameterChild(parameters []Parameter, globals []GlobalParameter, query string) *helpHintChild {
	search := searchText(query)
	count := 0
	var exact Parameter
	for _, parameter := range parameters {
		names := append([]string{parameter.Name, parameter.RawName}, parameter.Options...)
		if bestSearchRank(search, names...) != 1 {
			continue
		}
		count++
		exact = parameter
	}
	for _, global := range globals {
		if bestSearchRank(search, global.Name) == 1 {
			count++
		}
	}
	if count != 1 || !runtimeParameterHasNestedFields(exact) {
		return nil
	}
	return &helpHintChild{kind: helpHintParameter, parameter: strings.TrimLeft(firstOption(exact.Options, exact.Name), "-")}
}

func runtimeParameterHasNestedFields(parameter Parameter) bool {
	if len(parameter.Fields) > 0 {
		return true
	}
	return runtimeParameterShapeHasFields(parameter.Element) || runtimeParameterShapeHasFields(parameter.Value)
}

func runtimeParameterShapeHasFields(parameter *Parameter) bool {
	if parameter == nil {
		return false
	}
	if len(parameter.Fields) > 0 {
		return true
	}
	return runtimeParameterShapeHasFields(parameter.Element) || runtimeParameterShapeHasFields(parameter.Value)
}

func renderHelpHintFooter(w io.Writer, next *Next, language string, leadingBlank bool) error {
	if next == nil {
		return nil
	}
	if leadingBlank {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	textCommand := func(command string) string {
		if command == "" {
			return ""
		}
		command = strings.TrimSuffix(command, " --cli-output json")
		return command + " [--cli-output json]"
	}
	writeDefault := func(name, command string) error {
		if command == "" {
			return nil
		}
		_, err := fmt.Fprintf(w, "%s: %s\n", name, textCommand(command))
		return err
	}
	wroteBlock := false
	writeBlock := func(name, command string) error {
		if command == "" {
			return nil
		}
		if wroteBlock {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		wroteBlock = true
		punctuation := ":"
		if language == "zh" {
			punctuation = "："
		}
		_, err := fmt.Fprintf(w, "%s%s\n  %s\n", name, punctuation, textCommand(command))
		return err
	}

	switch next.operation {
	case "all":
		if err := writeBlock(label(language, "Search this Help", "搜索当前 Help"), next.Search); err != nil {
			return err
		}
	case "search":
		if err := writeBlock(label(language, "Try another keyword", "更换关键词重新搜索"), next.Search); err != nil {
			return err
		}
		if err := writeBlock(label(language, "Show all matches", "查看全部匹配"), next.SearchAll); err != nil {
			return err
		}
	default:
		if err := writeDefault(label(language, "Show all", "显示全部"), next.ShowAll); err != nil {
			return err
		}
		if err := writeDefault(label(language, "Search", "搜索"), next.Search); err != nil {
			return err
		}
	}

	childLabel := ""
	switch next.childKind {
	case helpHintAPI:
		childLabel = fmt.Sprintf(label(language, "Search parameters in %s", "继续搜索 %s 的参数"), next.childName)
	case helpHintParameter:
		childLabel = fmt.Sprintf(label(language, "Search nested fields in %s", "继续搜索 %s 的嵌套字段"), next.childName)
	}
	return writeBlock(childLabel, next.ChildSearch)
}

func attachRuntimeAIModeHint(document any, enabled bool) {
	var value *AIModeHint
	if enabled {
		value = &AIModeHint{Command: runtimeAIModeEnableCommand, Message: runtimeAIModeEnableMessage}
	}
	switch typed := document.(type) {
	case *ProductDocument:
		if typed != nil {
			typed.AIModeHint = value
		}
	case *ActionDocument:
		if typed != nil {
			typed.AIModeHint = value
		}
	case *RequestDocument:
		if typed != nil {
			typed.AIModeHint = value
		}
	case *APIParameterDocument:
		if typed != nil {
			typed.AIModeHint = value
		}
	case *APIResponseDocument:
		if typed != nil {
			typed.AIModeHint = value
		}
	}
}

func renderRuntimeTextFooter(w io.Writer, document any, options HelpOptions) error {
	if options.AIMode {
		return nil
	}
	operation := helpHintOperation(options)
	if next := runtimeDocumentNext(document); next != nil && next.operation != "" {
		operation = next.operation
	}
	if operation == "default" {
		target, ok := runtimeHelpHintTarget(document, options)
		if ok {
			command := buildHelpHintCommand(target, "default", "", false, true)
			if _, err := fmt.Fprintf(w, "\n%s\n  %s\n", label(options.Language,
				"For machine-readable Help, run:", "获取机器可读帮助，请运行："), command); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(w, "\n%s\n", runtimeAIModeEnableTextHint)
	return err
}

func runtimeDocumentNext(document any) *Next {
	switch typed := document.(type) {
	case *ProductDocument:
		if typed != nil {
			return typed.Next
		}
	case *ActionDocument:
		if typed != nil {
			return typed.Next
		}
	case *RequestDocument:
		if typed != nil {
			return typed.Next
		}
	case *APIParameterDocument:
		if typed != nil {
			return typed.Next
		}
	case *APIResponseDocument:
		if typed != nil {
			return typed.Next
		}
	}
	return nil
}

func normalizeRuntimeNextOutput(document any, includeJSON bool) {
	next := runtimeDocumentNext(document)
	if next == nil {
		return
	}
	normalize := func(command string) string {
		command = strings.TrimSuffix(command, " --cli-output json")
		if includeJSON && command != "" {
			command += " --cli-output json"
		}
		return command
	}
	next.ShowAll = normalize(next.ShowAll)
	next.Search = normalize(next.Search)
	next.SearchAll = normalize(next.SearchAll)
	next.ChildSearch = normalize(next.ChildSearch)
}

func runtimeHelpHintTarget(document any, options HelpOptions) (helpHintTarget, bool) {
	switch typed := document.(type) {
	case *ProductDocument:
		if typed == nil || typed.Target.Product == "" {
			return helpHintTarget{}, false
		}
		return helpHintTargetForProduct(typed, options), true
	case *ActionDocument:
		if typed == nil || typed.Target.Product == "" || typed.Target.API == "" {
			return helpHintTarget{}, false
		}
		options.ExplicitSection = false
		return helpHintTargetForAPI(typed.Target, options), true
	case *RequestDocument:
		if typed == nil || typed.Target.Product == "" || typed.Target.API == "" {
			return helpHintTarget{}, false
		}
		options.ExplicitSection = true
		options.Section = SectionRequest
		return helpHintTargetForAPI(typed.Target, options), true
	case *APIParameterDocument:
		if typed == nil || typed.Target.Product == "" || typed.Target.API == "" || typed.Parameter.Name == "" {
			return helpHintTarget{}, false
		}
		return helpHintTargetForParameter(typed.Target, typed.Parameter.Name, options), true
	case *APIResponseDocument:
		if typed == nil || typed.Target.Product == "" || typed.Target.API == "" {
			return helpHintTarget{}, false
		}
		options.ExplicitSection = true
		options.Section = SectionResponse
		return helpHintTargetForAPI(typed.Target, options), true
	default:
		return helpHintTarget{}, false
	}
}
