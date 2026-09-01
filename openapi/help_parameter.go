package openapi

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// renderParameterHelpText renders the same complete L3 object carried by JSON.
// It is deliberately independent from Action rendering and never applies the
// default 100-line budget.
func renderParameterHelpText(w io.Writer, document *machineHelpParameterDocument) error {
	if document == nil {
		return fmt.Errorf("parameter Help document is nil")
	}
	if document.Query != "" {
		if len(document.Matches) == 0 {
			if _, err := fmt.Fprintf(w, noHelpSearchMatchesFormat+"\n", document.Query); err != nil {
				return err
			}
			if document.Result == nil {
				return nil
			}
			return renderHelpProjectionResult(w, "matches", *document.Result, document.Next)
		}
		if _, err := fmt.Fprintln(w, "Matched fields:"); err != nil {
			return err
		}
		for _, match := range document.Matches {
			path := strings.Split(match.Path, ".")
			if err := renderMachineHelpParameterNode(w, match.Parameter, path, 2, true); err != nil {
				return err
			}
		}
		if document.Result == nil {
			return nil
		}
		return renderHelpProjectionResult(w, "matches", *document.Result, document.Next)
	}
	name := helpParameterDisplayName(document.Parameter)
	api := ""
	if len(document.Target.Path) >= 2 {
		api = document.Target.Path[len(document.Target.Path)-2]
	}
	if _, err := fmt.Fprintf(w, "%s: %s\n%s: %s\n%s: %s\n",
		machineHelpLabel("Parameter", "参数"), name,
		machineHelpLabel("API", "API"), api,
		machineHelpLabel("API Version", "API 版本"), document.Target.APIVersion,
	); err != nil {
		return err
	}
	rootName := firstNonEmptyMachineHelpString(document.Parameter.RawName, document.Parameter.Name)
	if err := renderMachineHelpParameterNode(w, document.Parameter, []string{rootName}, 0, false); err != nil {
		return err
	}
	if document.Result == nil {
		return renderHelpHintFooter(w, document.Next, true)
	}
	return renderHelpProjectionResult(w, "matches", *document.Result, document.Next)
}

func renderMachineHelpParameterNode(w io.Writer, parameter machineHelpParameter, path []string, indent int, printPath bool) error {
	prefix := strings.Repeat(" ", indent)
	if printPath {
		if _, err := fmt.Fprintf(w, "%s%s\n", prefix, strings.Join(path, ".")); err != nil {
			return err
		}
		indent += 2
		prefix = strings.Repeat(" ", indent)
	}
	write := func(label, value string) error {
		if value == "" {
			return nil
		}
		_, err := fmt.Fprintf(w, "%s%s: %s\n", prefix, label, value)
		return err
	}
	if err := write(machineHelpLabel("Type", "类型"), parameter.Type); err != nil {
		return err
	}
	if err := write(machineHelpLabel("Location", "位置"), parameter.Location); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s%s: %t\n", prefix, machineHelpLabel("Required", "必填"), parameter.Required); err != nil {
		return err
	}
	if description := localizedMachineHelpText(parameter.Help); description != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s:\n", prefix, machineHelpLabel("Description", "描述")); err != nil {
			return err
		}
		width := machineHelpMaxLineLength(w) - len([]rune(prefix)) - 2
		for _, line := range wrapMachineHelpText(description, width) {
			if line == "" {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s  %s\n", prefix, line); err != nil {
				return err
			}
		}
	}
	if parameter.Example != "" {
		if _, err := fmt.Fprintf(w, "\n%s%s:\n%s  %s\n", prefix, machineHelpLabel("Example", "示例"), prefix, parameter.Example); err != nil {
			return err
		}
	}
	if hasMachineHelpConstraints(parameter.Constraints) {
		if _, err := fmt.Fprintf(w, "%s%s:\n", prefix, machineHelpLabel("Constraints", "约束")); err != nil {
			return err
		}
		if err := renderMachineHelpConstraints(w, prefix+"  ", parameter.Constraints); err != nil {
			return err
		}
	}
	if len(parameter.Fields) > 0 {
		if _, err := fmt.Fprintf(w, "\n%s%s:\n", prefix, machineHelpLabel("Fields", "字段")); err != nil {
			return err
		}
		for _, field := range parameter.Fields {
			name := firstNonEmptyMachineHelpString(field.RawName, field.Name)
			if err := renderMachineHelpParameterNode(w, field, appendHelpParameterPath(path, name), indent+2, true); err != nil {
				return err
			}
		}
	}
	for _, entry := range []struct {
		label string
		shape *machineHelpShape
	}{{machineHelpLabel("Element", "元素"), parameter.Element}, {machineHelpLabel("Value", "值"), parameter.Value}} {
		if entry.shape == nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s%s:\n", prefix, entry.label); err != nil {
			return err
		}
		if err := renderMachineHelpShapeNode(w, entry.shape, path, indent+2); err != nil {
			return err
		}
	}
	return nil
}

func renderMachineHelpShapeNode(w io.Writer, shape *machineHelpShape, path []string, indent int) error {
	if shape == nil {
		return nil
	}
	prefix := strings.Repeat(" ", indent)
	if shape.Type != "" {
		if _, err := fmt.Fprintf(w, "%sType: %s\n", prefix, shape.Type); err != nil {
			return err
		}
	}
	if shape.Format != "" {
		if _, err := fmt.Fprintf(w, "%sFormat: %s\n", prefix, shape.Format); err != nil {
			return err
		}
	}
	if hasMachineHelpConstraints(shape.Constraints) {
		if _, err := fmt.Fprintf(w, "%s%s:\n", prefix, machineHelpLabel("Constraints", "约束")); err != nil {
			return err
		}
		if err := renderMachineHelpConstraints(w, prefix+"  ", shape.Constraints); err != nil {
			return err
		}
	}
	for _, field := range shape.Fields {
		name := firstNonEmptyMachineHelpString(field.RawName, field.Name)
		if err := renderMachineHelpParameterNode(w, field, appendHelpParameterPath(path, name), indent, true); err != nil {
			return err
		}
	}
	for _, entry := range []struct {
		label string
		shape *machineHelpShape
	}{{machineHelpLabel("Element", "元素"), shape.Element}, {machineHelpLabel("Value", "值"), shape.Value}} {
		if entry.shape == nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s%s:\n", prefix, entry.label); err != nil {
			return err
		}
		if err := renderMachineHelpShapeNode(w, entry.shape, path, indent+2); err != nil {
			return err
		}
	}
	return nil
}

func renderMachineHelpConstraints(w io.Writer, prefix string, constraints machineHelpConstraints) error {
	entries := []struct{ label, value string }{
		{machineHelpLabel("Enum", "枚举"), strings.Join(constraints.Enum, ", ")},
		{machineHelpLabel("Pattern", "模式"), constraints.Pattern},
		{machineHelpLabel("Minimum", "最小值"), constraints.Minimum},
		{machineHelpLabel("Maximum", "最大值"), constraints.Maximum},
		{machineHelpLabel("Minimum length", "最小长度"), constraints.MinLength},
		{machineHelpLabel("Maximum length", "最大长度"), constraints.MaxLength},
	}
	for _, entry := range entries {
		if entry.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s%s: %s\n", prefix, entry.label, entry.value); err != nil {
			return err
		}
	}
	return nil
}

func hasMachineHelpConstraints(constraints machineHelpConstraints) bool {
	return len(constraints.Enum) > 0 || constraints.Pattern != "" || constraints.Minimum != "" ||
		constraints.Maximum != "" || constraints.MinLength != "" || constraints.MaxLength != ""
}

// machineHelpParameterDocument is the stable L3 document shared by default,
// all, search, Text and JSON renderers. Parameter always remains the complete
// finite tree; Search results are an orthogonal projection.
type machineHelpParameterDocument struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Kind          string                      `json:"helpLevel"`
	Target        machineHelpTarget           `json:"-"`
	Query         string                      `json:"query"`
	Parameter     machineHelpParameter        `json:"parameter"`
	Matches       []machineHelpParameterMatch `json:"matches"`
	Result        *HelpResult                 `json:"result"`
	Next          *HelpNext                   `json:"next,omitempty"`
	AIModeHint    *machineHelpAIModeHint      `json:"aiModeHint"`
}

type machineHelpParameterMatch struct {
	Kind      string               `json:"kind"`
	Path      string               `json:"path"`
	Parameter machineHelpParameter `json:"parameter"`
	Rank      HelpSearchRank       `json:"rank"`
}

// buildParameterHelpDocument resolves only executable top-level flags in the
// active style plus public globals. Nested paths are documentation fields and
// intentionally cannot be resolved as L3 command-line flags.
func buildParameterHelpDocument(action *machineHelpAPIDocument, option string) (*machineHelpParameterDocument, error) {
	if action == nil {
		return nil, fmt.Errorf("action Help document is nil")
	}
	option = strings.TrimSpace(option)
	if option == "" {
		return nil, fmt.Errorf("parameter Help requires a flag")
	}

	candidates := append([]machineHelpParameter(nil), activeMachineHelpParameters(action)...)
	candidates = append(candidates, action.GlobalParameters...)
	matches := make([]machineHelpParameter, 0, 1)
	for _, parameter := range candidates {
		for _, formation := range parameter.Options {
			if formation == option {
				matches = append(matches, parameter)
				break
			}
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("unknown parameter flag %q", option)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("parameter flag %q is ambiguous", option)
	}

	target := machineHelpTarget{
		Path:           append(append([]string(nil), action.Target.Path...), option),
		RequestedStyle: action.Target.RequestedStyle,
		APIVersion:     action.API.Operation.APIVersion,
	}
	return &machineHelpParameterDocument{
		SchemaVersion: action.SchemaVersion,
		Kind:          "parameter",
		Target:        target,
		Parameter:     matches[0],
	}, nil
}

// searchParameterHelpDocument uses the shared tokenizer/ranker over the root
// parameter and every finite fields/element/value descendant, then applies the
// global twenty-result Search limit after full ranking.
func searchParameterHelpDocument(document *machineHelpParameterDocument, query string, unlimited bool) {
	if document == nil {
		return
	}
	document.Query = strings.TrimSpace(query)
	type parameterPathValue struct {
		path      string
		parameter machineHelpParameter
	}
	candidates := make([]HelpSearchCandidate, 0)
	walkMachineHelpParameter(document.Parameter, nil, func(path string, parameter machineHelpParameter) {
		aliases := append([]string{parameter.RawName, path}, parameter.Options...)
		documentText := []string{
			parameter.Help.EN,
			parameter.Help.ZH,
			parameter.Example,
			strings.Join(parameter.Constraints.Enum, " "),
			parameter.Constraints.Pattern,
			parameter.Constraints.Minimum,
			parameter.Constraints.Maximum,
			parameter.Constraints.MinLength,
			parameter.Constraints.MaxLength,
		}
		candidates = append(candidates, HelpSearchCandidate{
			Kind:          "parameter-field",
			Name:          helpParameterDisplayName(parameter),
			Aliases:       aliases,
			DescriptionEN: strings.Join(documentText, "\n"),
			DescriptionZH: strings.Join(documentText, "\n"),
			Value:         parameterPathValue{path: path, parameter: parameter},
		})
	})

	projected := ProjectHelpSearchMatches(SearchHelpCandidates(candidates, query), unlimited)
	document.Matches = make([]machineHelpParameterMatch, 0, len(projected.Matches))
	for _, match := range projected.Matches {
		value, ok := match.Candidate.Value.(parameterPathValue)
		if !ok {
			continue
		}
		document.Matches = append(document.Matches, machineHelpParameterMatch{
			Kind:      "parameter-field",
			Path:      value.path,
			Parameter: value.parameter,
			Rank:      match.Rank,
		})
	}
	document.Result = &projected.Result
}

func walkMachineHelpParameter(parameter machineHelpParameter, parent []string, visit func(string, machineHelpParameter)) {
	name := parameter.RawName
	if name == "" {
		name = parameter.Name
	}
	path := appendHelpParameterPath(parent, name)
	visit(strings.Join(path, "."), parameter)
	for _, field := range parameter.Fields {
		walkMachineHelpParameter(field, path, visit)
	}
	for _, shape := range []*machineHelpShape{parameter.Element, parameter.Value} {
		walkMachineHelpShape(shape, path, visit)
	}
}

func walkMachineHelpShape(shape *machineHelpShape, parent []string, visit func(string, machineHelpParameter)) {
	if shape == nil {
		return
	}
	for _, field := range shape.Fields {
		walkMachineHelpParameter(field, parent, visit)
	}
	walkMachineHelpShape(shape.Element, parent, visit)
	walkMachineHelpShape(shape.Value, parent, visit)
}

func appendHelpParameterPath(parent []string, name string) []string {
	result := append([]string(nil), parent...)
	if name != "" {
		result = append(result, name)
	}
	return result
}

func helpParameterDisplayName(parameter machineHelpParameter) string {
	if len(parameter.Options) > 0 {
		options := append([]string(nil), parameter.Options...)
		sort.Strings(options)
		return options[0]
	}
	if parameter.Name != "" {
		return parameter.Name
	}
	return parameter.RawName
}
