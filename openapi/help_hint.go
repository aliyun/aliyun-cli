package openapi

import (
	"fmt"
	"io"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

type helpHintChildKind uint8

const (
	helpHintChildNone helpHintChildKind = iota
	helpHintChildProduct
	helpHintChildAPI
	helpHintChildParameter
)

// helpHintExact records only navigation evidence. Search selection, ranking,
// ordering and limiting remain owned by help_search.go.
type helpHintExact struct {
	count     int
	kind      helpHintChildKind
	product   string
	action    string
	parameter string
}

func (h *helpHintExact) observe(match HelpSearchMatch, kind helpHintChildKind, product, action, parameter string) {
	if match.Rank != HelpSearchExactName {
		return
	}
	h.count++
	if h.count == 1 {
		h.kind = kind
		h.product = product
		h.action = action
		h.parameter = parameter
		return
	}
	// More than one exact result is deliberately non-navigable, even if only
	// one of those results would otherwise have children.
	h.kind = helpHintChildNone
	h.product = ""
	h.action = ""
	h.parameter = ""
}

func setRootHelpNext(document *machineHelpRootDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, document.Result.Truncated) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
	attachHelpChildSearch(document.Next, target, aiMode, document.helpHintExact)
}

func setProductHelpNext(document *machineHelpProductDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, document.Result.Truncated) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
	attachHelpChildSearch(document.Next, target, aiMode, document.helpHintExact)
}

func setActionHelpNext(document *machineHelpAPIDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, document.Result.Truncated) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
	attachHelpChildSearch(document.Next, target, aiMode, document.helpHintExact)
}

func setParameterHelpNext(document *machineHelpParameterDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, helpResultTruncated(document.Result)) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func setResponseHelpNext(document *machineHelpAPIResponseDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, document.Result.Truncated) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func setUtilityHelpNext(document *machineHelpUtilityDocument, target HelpTarget, aiMode bool) {
	if document == nil || !helpHintNavigationApplies(target, document.Result.Truncated) {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func helpResultTruncated(result *HelpResult) bool {
	return result != nil && result.Truncated
}

func helpHintNavigationApplies(target HelpTarget, defaultEligible bool) bool {
	switch target.Operation {
	case HelpOperationAll, HelpOperationSearch:
		return true
	default:
		return defaultEligible
	}
}

func buildHelpNext(target HelpTarget, aiMode bool) *HelpNext {
	command := func(nextTarget HelpTarget) string {
		if aiMode {
			nextTarget.Output = HelpOutputText
		}
		value, _ := BuildHelpCommand(nextTarget)
		return value
	}

	search := target
	search.Operation = HelpOperationSearch
	search.SearchQuery = "<keyword>"
	search.SearchAll = false
	next := &HelpNext{Search: command(search), operation: target.Operation}

	switch target.Operation {
	case HelpOperationDefault:
		all := target
		all.Operation = HelpOperationAll
		all.SearchQuery = ""
		all.SearchAll = false
		next.ShowAll = command(all)
	case HelpOperationSearch:
		if !target.SearchAll {
			searchAll := target
			searchAll.Operation = HelpOperationSearch
			searchAll.SearchAll = true
			next.SearchAll = command(searchAll)
		}
	}
	return next
}

func attachHelpChildSearch(next *HelpNext, parent HelpTarget, aiMode bool, exact helpHintExact) {
	if next == nil || parent.Operation != HelpOperationSearch || exact.count != 1 || exact.kind == helpHintChildNone {
		return
	}
	child := parent
	child.Section = HelpSectionRequest
	child.SectionExplicit = false
	child.Operation = HelpOperationSearch
	child.SearchQuery = "<keyword>"
	child.SearchAll = false
	switch exact.kind {
	case helpHintChildProduct:
		child.Level = HelpLevelProduct
		child.Product = exact.product
		child.Action = ""
		child.Parameter = ""
		child.Version = ""
		child.VersionFlag = ""
		if child.CommandStyle == "" {
			child.CommandStyle = CommandStyleCamel
		}
	case helpHintChildAPI:
		child.Level = HelpLevelAction
		child.Action = exact.action
		child.Parameter = ""
	case helpHintChildParameter:
		child.Level = HelpLevelParameter
		child.Parameter = exact.parameter
	default:
		return
	}
	if aiMode {
		child.Output = HelpOutputText
	}
	command, err := BuildHelpCommand(child)
	if err != nil {
		return
	}
	next.ChildSearch = command
	next.childKind = exact.kind
	switch exact.kind {
	case helpHintChildProduct:
		next.childName = exact.product
	case helpHintChildAPI:
		next.childName = exact.action
	case helpHintChildParameter:
		next.childName = strings.TrimLeft(exact.parameter, "-")
	}
}

func renderHelpHintFooter(w io.Writer, next *HelpNext, leadingBlank bool) error {
	if next == nil {
		return nil
	}
	if leadingBlank {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	writeDefault := func(label, command string) error {
		if command == "" {
			return nil
		}
		_, err := fmt.Fprintf(w, "%s: %s\n", label, helpHintTextCommand(command))
		return err
	}
	wroteBlock := false
	writeBlock := func(label, command string) error {
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
		if i18n.GetLanguage() == "zh" {
			punctuation = "："
		}
		_, err := fmt.Fprintf(w, "%s%s\n  %s\n", label, punctuation, helpHintTextCommand(command))
		return err
	}

	switch next.operation {
	case HelpOperationAll:
		if err := writeBlock(helpHintLabel("Search this Help", "搜索当前 Help"), next.Search); err != nil {
			return err
		}
	case HelpOperationSearch:
		if err := writeBlock(helpHintLabel("Try another keyword", "更换关键词重新搜索"), next.Search); err != nil {
			return err
		}
		if err := writeBlock(helpHintLabel("Show all matches", "查看全部匹配"), next.SearchAll); err != nil {
			return err
		}
	default:
		if err := writeDefault("Show all", next.ShowAll); err != nil {
			return err
		}
		if err := writeDefault("Search", next.Search); err != nil {
			return err
		}
	}

	childLabel := ""
	switch next.childKind {
	case helpHintChildProduct:
		childLabel = fmt.Sprintf(helpHintLabel("Search APIs in %s", "继续搜索 %s 的 API"), next.childName)
	case helpHintChildAPI:
		childLabel = fmt.Sprintf(helpHintLabel("Search parameters in %s", "继续搜索 %s 的参数"), next.childName)
	case helpHintChildParameter:
		childLabel = fmt.Sprintf(helpHintLabel("Search nested fields in %s", "继续搜索 %s 的嵌套字段"), next.childName)
	}
	return writeBlock(childLabel, next.ChildSearch)
}

func helpHintLabel(english, chinese string) string {
	return i18n.T(english, chinese).Text()
}

func helpHintTextCommand(command string) string {
	if strings.TrimSpace(command) == "" {
		return ""
	}
	return command + " [--cli-output json]"
}

func renderMachineReadableHelpHint(w io.Writer, target HelpTarget) error {
	target.Output = HelpOutputJSON
	command, err := BuildHelpCommand(target)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "\n%s\n  %s\n", helpHintLabel(
		"For machine-readable Help, run:",
		"获取机器可读帮助，请运行：",
	), command)
	return err
}

func utilityHelpTarget(path []string, options cli.HelpOptions) HelpTarget {
	target := HelpTarget{Level: HelpLevelUtility, Operation: HelpOperationDefault, Output: HelpOutput(options.Output)}
	if len(path) > 1 {
		target.Utility = path[1]
	}
	if options.Operation != "" {
		target.Operation = HelpOperation(options.Operation)
	}
	target.SearchQuery = options.SearchQuery
	target.SearchAll = options.SearchAll
	return target
}

func preferredHelpParameterOption(parameter machineHelpParameter) string {
	if len(parameter.Options) > 0 {
		return strings.TrimLeft(parameter.Options[0], "-")
	}
	return strings.TrimLeft(firstNonEmptyMachineHelpString(parameter.RawName, parameter.Name), "-")
}

func machineHelpParameterHasNestedFields(parameter machineHelpParameter) bool {
	if len(parameter.Fields) > 0 {
		return true
	}
	return machineHelpShapeHasFields(parameter.Element) || machineHelpShapeHasFields(parameter.Value)
}

func machineHelpShapeHasFields(shape *machineHelpShape) bool {
	if shape == nil {
		return false
	}
	if len(shape.Fields) > 0 {
		return true
	}
	return machineHelpShapeHasFields(shape.Element) || machineHelpShapeHasFields(shape.Value)
}
