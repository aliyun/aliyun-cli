package help

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

type TextRenderer struct{}
type JSONRenderer struct{}

func Render(w io.Writer, document any, options HelpOptions) error {
	options = options.normalized()
	if options.Format == FormatJSON {
		return (JSONRenderer{}).Render(w, document, options)
	}
	return (TextRenderer{}).Render(w, document, options)
}

func (JSONRenderer) Render(w io.Writer, document any, options HelpOptions) error {
	if w == nil {
		return fmt.Errorf("help output is nil")
	}
	options = options.normalized()
	attachRuntimeAIModeHint(document, !options.AIMode)
	normalizeRuntimeNextOutput(document, !options.AIMode)
	projected, err := localizeDocumentRawJSON(document, options.Language)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	if !options.AIMode {
		encoder.SetIndent("", "  ")
	}
	encoder.SetEscapeHTML(false)
	return encoder.Encode(projected)
}

func (TextRenderer) Render(w io.Writer, document any, options HelpOptions) error {
	if w == nil {
		return fmt.Errorf("help output is nil")
	}
	options = options.normalized()
	var err error
	switch typed := document.(type) {
	case *ProductDocument:
		err = renderProductText(w, typed, options.Language)
	case *ActionDocument:
		err = renderActionText(w, typed, options.Language)
	case *RequestDocument:
		err = renderRequestText(w, typed, options.Language)
	case *APIParameterDocument:
		err = renderParameterDocumentText(w, typed, options.Language)
	case *APIResponseDocument:
		err = renderResponseText(w, typed, options.Language)
	default:
		return fmt.Errorf("unsupported Runtime Help v1 document %T", document)
	}
	if err != nil {
		return err
	}
	return renderRuntimeTextFooter(w, document, options)
}

func renderProvenanceText(w io.Writer, provenance *MetadataProvenance, language string) {
	if provenance == nil {
		return
	}
	provider := provenance.Kind
	if provenance.PluginName != "" {
		provider = provenance.PluginName
		if provenance.PluginVersion != "" {
			provider += " (" + provenance.PluginVersion + ")"
		}
	}
	fmt.Fprintf(w, "%s: %s\n", label(language, "Metadata provider", "元数据来源"), provider)
}

func renderParameterDocumentText(w io.Writer, document *APIParameterDocument, language string) error {
	if document == nil {
		return fmt.Errorf("parameter Help document is nil")
	}
	fmt.Fprintf(w, "%s: --%s\n", label(language, "Parameter", "参数"), strings.TrimLeft(document.Parameter.Name, "-"))
	fmt.Fprintf(w, "%s: %s\n", label(language, "API", "API"), document.Command)
	fmt.Fprintf(w, "%s: %s\n\n", label(language, "API Version", "API 版本"), document.Target.APIVersion)
	renderProvenanceText(w, document.Provenance, language)
	if document.Query != "" {
		if len(document.Matches) == 0 {
			fmt.Fprintf(w, "%s\n", fmt.Sprintf(
				label(language, "No parameter entries matched %q.", "没有参数条目匹配 %q。"),
				document.Query))
		} else {
			fmt.Fprintf(w, "%s:\n", label(language, "Matched parameter paths", "匹配的参数路径"))
			for _, match := range document.Matches {
				fmt.Fprintf(w, "\n  %s\n", match.Path)
				renderParameterText(w, match.Parameter, language, "    ", true)
			}
		}
		return renderResult(w, document.Result, language, document.Next)
	}
	renderParameterText(w, document.Parameter, language, "", true)
	return renderResult(w, document.Result, language, document.Next)
}

func renderProductText(w io.Writer, document *ProductDocument, language string) error {
	if document == nil {
		return fmt.Errorf("product Help document is nil")
	}
	name := document.Product.Name.Text(language)
	fmt.Fprintf(w, "%s: %s", label(language, "Product", "产品"), document.Product.Code)
	if name != "" {
		fmt.Fprintf(w, " (%s)", name)
	}
	fmt.Fprintf(w, "\n%s: %s\n", label(language, "API Version", "API 版本"), document.Product.SelectedVersion)
	if description := document.Product.Description.Text(language); description != "" {
		fmt.Fprintf(w, "%s: %s\n", label(language, "Description", "描述"), description)
	}
	renderProvenanceText(w, document.Provenance, language)
	fmt.Fprintf(w, "\n%s:\n", label(language, "Available APIs", "可用 API"))
	for _, api := range document.APIs {
		name := api.Command
		description := api.Title.Text(language)
		if description == "" {
			description = api.Description.Text(language)
		}
		if api.Deprecated {
			description = label(language, "[Deprecated] ", "[已废弃] ") + description
		}
		fmt.Fprintf(w, "  %-30s %s\n", name, description)
	}
	if document.Result.OmittedDeprecated > 0 {
		fmt.Fprintf(w, "\n%s\n", fmt.Sprintf(
			label(language, "Omitting %d deprecated APIs; use --help-all to include them.", "已省略 %d 个废弃 API；使用 --help-all 显示。"),
			document.Result.OmittedDeprecated))
	}
	return renderResult(w, document.Result, language, document.Next)
}

func renderActionText(w io.Writer, document *ActionDocument, language string) error {
	if document == nil {
		return fmt.Errorf("action Help document is nil")
	}
	description := document.Title.Text(language)
	if description == "" {
		description = document.Description.Text(language)
	}
	fmt.Fprintf(w, "%s: %s\n", label(language, "Description", "描述"), description)
	fmt.Fprintf(w, "%s: %s\n", label(language, "API Version", "API 版本"), document.Target.APIVersion)
	renderProvenanceText(w, document.Provenance, language)
	fmt.Fprintf(w, "\n%s:\n  aliyun %s %s [parameters]\n", label(language, "Usage", "使用"), document.Target.Product, document.Command)
	fmt.Fprintf(w, "\n%s:\n", label(language, "Parameters", "参数"))
	for _, parameter := range document.Parameters {
		renderParameterText(w, parameter, language, "  ", false)
	}
	renderGlobalParametersText(w, document.GlobalParameters, language)
	if err := renderQueryOptionsText(w, document.QueryOptions, language, document.ResponseQuery); err != nil {
		return err
	}
	if example := document.Examples.Kebab; example != "" {
		fmt.Fprintf(w, "\n%s:\n  %s\n", label(language, "Example", "示例"), example)
	}
	return renderResult(w, document.Result, language, document.Next)
}

func renderRequestText(w io.Writer, document *APIRequestDocument, language string) error {
	if document == nil {
		return fmt.Errorf("request Help document is nil")
	}
	fmt.Fprintf(w, "%s: %s\n", label(language, "Description", "描述"), document.Description.Text(language))
	fmt.Fprintf(w, "%s: %s\n", label(language, "API Version", "API 版本"), document.Target.APIVersion)
	renderProvenanceText(w, document.Provenance, language)
	fmt.Fprintf(w, "\n%s:\n  aliyun %s %s [parameters]\n", label(language, "Usage", "使用"), document.Target.Product, document.Command)
	fmt.Fprintf(w, "\n%s:\n", label(language, "Parameters", "参数"))
	for _, parameter := range document.Parameters {
		renderParameterText(w, parameter, language, "  ", false)
	}
	renderGlobalParametersText(w, document.GlobalParameters, language)
	if err := renderQueryOptionsText(w, document.QueryOptions, language, document.ResponseQuery); err != nil {
		return err
	}
	if example := document.Examples.Kebab; example != "" {
		fmt.Fprintf(w, "\n%s:\n  %s\n", label(language, "Example", "示例"), example)
	}
	return renderResult(w, document.Result, language, document.Next)
}

func renderGlobalParametersText(w io.Writer, parameters []GlobalParameter, language string) {
	if len(parameters) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s:\n", label(language, "Global Parameters", "全局参数"))
	for _, parameter := range parameters {
		fmt.Fprintf(w, "  %-28s %s", parameter.Name, parameter.Type)
		if text := parameter.Help.Text(language); text != "" {
			fmt.Fprintf(w, "  %s", text)
		}
		fmt.Fprintln(w)
	}
}

func renderQueryOptionsText(w io.Writer, options []QueryOption, language string, example *QueryExample) error {
	if len(options) == 0 {
		return nil
	}
	fmt.Fprintf(w, "\n%s:\n", label(language, "Query Options", "查询选项"))
	for _, option := range options {
		defaultText := ""
		if option.HasDefault {
			defaultText = ", " + label(language, "default", "默认") + ": " + option.Default
		}
		fmt.Fprintf(w, "  %-28s %s%s  %s\n", option.Name, option.Type, defaultText, option.Help.Text(language))
	}
	if example != nil {
		fmt.Fprintf(w, "\n  %s (JMESPath: %s):\n    1. %s\n      %s\n    2. %s\n      %s\n",
			label(language, "Response aggregation example", "响应聚合示例"), example.Path,
			label(language, "Inspect the response structure", "查看响应结构"), example.SchemaCommand,
			label(language, "Query only the required fields", "仅查询所需字段"), example.QueryCommand)
	}
	return nil
}

func renderParameterText(w io.Writer, parameter Parameter, language, indent string, expanded bool) {
	typeText := displayParameterType(parameter.Type)
	if parameter.Required {
		typeText += " (" + label(language, "required", "必填") + ")"
	}
	namePrefix := fmt.Sprintf("%s--%-26s ", indent, strings.TrimLeft(parameter.Name, "-"))
	typeColumn := len([]rune(namePrefix))
	prefix := namePrefix + typeText + ","
	descriptionWidth := helpMaxLineLength() - len([]rune(prefix)) - 2
	writeHelpWrappedWithWidth(w, prefix, parameter.Help.Text(language), typeColumn, descriptionWidth)
	renderParameterDetails(w, parameter, language, strings.Repeat(" ", typeColumn))
	if !expanded {
		return
	}
	for _, field := range parameter.Fields {
		renderParameterText(w, field, language, indent+"  ", true)
	}
	if parameter.Element != nil {
		fmt.Fprintf(w, "%s  %s:\n", indent, label(language, "Element", "元素"))
		renderParameterShapeText(w, *parameter.Element, language, indent+"    ")
	}
	if parameter.Value != nil {
		fmt.Fprintf(w, "%s  %s:\n", indent, label(language, "Value", "值"))
		renderParameterShapeText(w, *parameter.Value, language, indent+"    ")
	}
}

func displayParameterType(dataType meta.DataType) string {
	switch dataType {
	case meta.TypeMap:
		return "string"
	case meta.TypeArray:
		return "list"
	case meta.TypeInteger, meta.TypeLong:
		return "int"
	case meta.TypeBoolean:
		return "bool"
	default:
		return string(dataType)
	}
}

func renderParameterDetails(w io.Writer, parameter Parameter, language, indent string) {
	if parameter.Example != "" {
		fmt.Fprintf(w, "%s%s: %s\n", indent, label(language, "example", "示例"), parameter.Example)
	}
	var constraints []string
	if len(parameter.Constraints.Enum) > 0 {
		constraints = append(constraints, "enum=["+strings.Join(parameter.Constraints.Enum, ", ")+"]")
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{"pattern", parameter.Constraints.Pattern},
		{"minimum", parameter.Constraints.Minimum},
		{"maximum", parameter.Constraints.Maximum},
		{"minLength", parameter.Constraints.MinLength},
		{"maxLength", parameter.Constraints.MaxLength},
	} {
		if item.value != "" {
			constraints = append(constraints, item.name+"="+item.value)
		}
	}
	if len(constraints) > 0 {
		writeHelpWrapped(w, indent+label(language, "constraints", "约束")+":", strings.Join(constraints, ", "), len([]rune(indent))+2)
	}
	structure, format := parameterFormatHints(parameter)
	if structure != "" {
		writeHelpWrapped(w, indent+label(language, "structure", "结构")+":", structure, len([]rune(indent))+2)
	}
	if format != "" {
		writeHelpWrapped(w, indent+label(language, "format", "格式")+":", format, len([]rune(indent))+2)
	}
}

func parameterFormatHints(parameter Parameter) (structure, format string) {
	flag := strings.TrimLeft(parameter.Name, "-")
	switch parameter.Type {
	case meta.TypeArray:
		if parameter.Element == nil || isPrimitiveParameter(*parameter.Element) {
			return "", fmt.Sprintf("--%s value1 value2 value3", flag)
		}
		if parameter.Element.Type == meta.TypeObject && isFlatObjectParameter(*parameter.Element) {
			keys := sortedParameterFieldNames(*parameter.Element)
			return parameterShape(*parameter.Element), fmt.Sprintf(
				"--%s %s --%s %s",
				flag, parameterKVExample(keys, []string{"a", "b", "c", "d"}),
				flag, parameterKVExample(keys, []string{"e", "f", "g", "h"}),
			)
		}
		return parameterShape(parameter), fmt.Sprintf("--%s 'value'", flag)
	case meta.TypeObject:
		structure = parameterShape(parameter)
		if isFlatObjectParameter(parameter) {
			return structure, fmt.Sprintf("--%s %s ...", flag,
				parameterKVExample(sortedParameterFieldNames(parameter), []string{"xxx"}))
		}
		return structure, fmt.Sprintf("--%s 'value'", flag)
	case meta.TypeMap:
		structure = parameterShape(parameter)
		if parameter.Value == nil || isPrimitiveParameter(*parameter.Value) {
			return structure, fmt.Sprintf("--%s key1=value1 key2=value2 ...", flag)
		}
		return structure, fmt.Sprintf("--%s 'value'", flag)
	default:
		return "", ""
	}
}

func isPrimitiveParameter(parameter Parameter) bool {
	switch parameter.Type {
	case meta.TypeArray, meta.TypeObject, meta.TypeMap:
		return false
	default:
		return true
	}
}

func isFlatObjectParameter(parameter Parameter) bool {
	if parameter.Type != meta.TypeObject || len(parameter.Fields) == 0 {
		return false
	}
	for _, field := range parameter.Fields {
		if !isPrimitiveParameter(field) {
			return false
		}
	}
	return true
}

func sortedParameterFieldNames(parameter Parameter) []string {
	names := make([]string, 0, len(parameter.Fields))
	for _, field := range parameter.Fields {
		names = append(names, strings.TrimLeft(field.Name, "-"))
	}
	sort.Strings(names)
	return names
}

func parameterKVExample(keys, values []string) string {
	parts := make([]string, 0, len(keys))
	for index, key := range keys {
		value := "xxx"
		if len(values) > 0 {
			value = values[index%len(values)]
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, " ")
}

func parameterStructure(parameter Parameter) string {
	switch parameter.Type {
	case "object":
		if len(parameter.Fields) == 0 {
			return "{<key>=<value>, ...}"
		}
		parts := make([]string, 0, len(parameter.Fields))
		for _, field := range parameter.Fields {
			parts = append(parts, strings.TrimLeft(field.Name, "-")+": "+parameterShape(field))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case "array":
		if parameter.Element != nil {
			return "[" + parameterShape(*parameter.Element) + ", ...]"
		}
	case "map":
		if parameter.Value != nil {
			return "{<key>: " + parameterShape(*parameter.Value) + ", ...}"
		}
		return "{<key>: <value>, ...}"
	}
	return ""
}

func parameterShape(parameter Parameter) string {
	if structure := parameterStructure(parameter); structure != "" {
		return structure
	}
	return displayParameterType(parameter.Type)
}

func writeHelpWrapped(w io.Writer, prefix, text string, continuationIndent int) {
	writeHelpWrappedWithWidth(w, prefix, text, continuationIndent, helpMaxLineLength()-continuationIndent)
}

func writeHelpWrappedWithWidth(w io.Writer, prefix, text string, continuationIndent, width int) {
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(w, prefix)
		return
	}
	if width < 20 {
		width = 20
	}
	lines := wrapHelpText(text, width)
	if len([]rune(prefix))+2+len([]rune(lines[0])) <= helpMaxLineLength() {
		fmt.Fprintf(w, "%s  %s\n", prefix, lines[0])
		lines = lines[1:]
	} else {
		fmt.Fprintln(w, prefix)
	}
	indent := strings.Repeat(" ", continuationIndent)
	for _, line := range lines {
		fmt.Fprintf(w, "%s%s\n", indent, line)
	}
}

func helpMaxLineLength() int {
	if value := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH")); value != "" {
		if width, err := strconv.Atoi(value); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

func wrapHelpText(text string, width int) []string {
	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		runes := []rune(strings.TrimSpace(paragraph))
		for len(runes) > width {
			end := width
			for i := width - 1; i > width/2; i-- {
				if strings.ContainsRune(" ,.;/，。、、：", runes[i]) {
					end = i + 1
					break
				}
			}
			result = append(result, strings.TrimSpace(string(runes[:end])))
			runes = []rune(strings.TrimSpace(string(runes[end:])))
		}
		if len(runes) > 0 {
			result = append(result, string(runes))
		}
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func renderParameterShapeText(w io.Writer, shape Parameter, language, indent string) {
	fmt.Fprintf(w, "%s%s\n", indent, shape.Type)
	for _, field := range shape.Fields {
		renderParameterText(w, field, language, indent+"  ", true)
	}
	if shape.Element != nil {
		fmt.Fprintf(w, "%s  %s:\n", indent, label(language, "Element", "元素"))
		renderParameterShapeText(w, *shape.Element, language, indent+"    ")
	}
	if shape.Value != nil {
		fmt.Fprintf(w, "%s  %s:\n", indent, label(language, "Value", "值"))
		renderParameterShapeText(w, *shape.Value, language, indent+"    ")
	}
}

func renderResponseText(w io.Writer, document *APIResponseDocument, language string) error {
	if document == nil {
		return fmt.Errorf("response Help document is nil")
	}
	renderProvenanceText(w, document.Provenance, language)
	if len(document.Warnings) > 0 {
		fmt.Fprintf(w, "%s:\n", label(language, "Warnings", "警告"))
		for _, warning := range document.Warnings {
			fmt.Fprintf(w, "  - %s\n", warning)
		}
	}
	if notice := document.Notice.Text(language); notice != "" {
		fmt.Fprintln(w, notice)
	}
	if len(document.Matches) > 0 {
		fmt.Fprintf(w, "%s:\n", label(language, "Matched response paths", "匹配的响应路径"))
		for _, path := range document.Matches {
			fmt.Fprintf(w, "- %s\n", path)
		}
		fmt.Fprintln(w)
	}
	if document.OutputSchema != nil {
		fmt.Fprintf(w, "%s", label(language, "Response schema", "响应结构"))
		if document.OutputSchema.StatusCode != "" {
			fmt.Fprintf(w, " (HTTP %s)", document.OutputSchema.StatusCode)
		}
		fmt.Fprintln(w, ":")
		raw, err := LocalizeRawJSON(document.OutputSchema.Schema, language)
		if err != nil {
			return err
		}
		if err := writePrettyJSON(w, raw); err != nil {
			return err
		}
		if len(document.OutputSchema.Components) > 0 {
			fmt.Fprintf(w, "\n%s:\n", label(language, "Components", "组件"))
			localized, err := LocalizeRawJSONMap(document.OutputSchema.Components, language)
			if err != nil {
				return err
			}
			raw, _ = json.Marshal(map[string]any{"schemas": localized})
			if err := writePrettyJSON(w, raw); err != nil {
				return err
			}
		}
	} else if len(document.Responses) > 0 {
		fmt.Fprintf(w, "%s:\n", label(language, "Responses", "响应"))
		raw, err := LocalizeRawJSON(document.Responses, language)
		if err != nil {
			return err
		}
		if err := writePrettyJSON(w, raw); err != nil {
			return err
		}
		if len(document.Components) > 0 {
			fmt.Fprintf(w, "\n%s:\n", label(language, "Components", "组件"))
			localized, err := LocalizeRawJSONMap(document.Components, language)
			if err != nil {
				return err
			}
			raw, _ = json.Marshal(map[string]any{"schemas": localized})
			if err := writePrettyJSON(w, raw); err != nil {
				return err
			}
		}
	}
	if document.ResponseQuery != nil {
		fmt.Fprintf(w, "\n%s:\n  %s\n", label(language, "Query with JMESPath", "使用 JMESPath 查询"), document.ResponseQuery.QueryCommand)
	}
	return renderResult(w, document.Result, language, document.Next)
}

func renderResult(w io.Writer, result Result, language string, nextValues ...*Next) error {
	if result.Truncated {
		if _, err := fmt.Fprintf(w, "\n%s\n",
			fmt.Sprintf(label(language, "Showing %d of %d entries; use all mode for the complete document.", "显示 %d/%d 项；使用 all 模式查看完整文档。"), result.Shown, result.Total)); err != nil {
			return err
		}
	}
	if len(nextValues) == 0 {
		return nil
	}
	return renderHelpHintFooter(w, nextValues[0], language, !result.Truncated)
}

func label(language, en, zh string) string {
	if language == "zh" {
		return zh
	}
	return en
}

func writePrettyJSON(w io.Writer, raw json.RawMessage) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		_, err := fmt.Fprintln(w, "{}")
		return err
	}
	var output bytes.Buffer
	if err := json.Indent(&output, raw, "", "  "); err != nil {
		return err
	}
	output.WriteByte('\n')
	_, err := w.Write(output.Bytes())
	return err
}

// LocalizeRawJSON projects description/title *_en and *_zh pairs recursively.
func LocalizeRawJSON(raw json.RawMessage, language string) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	value = localizeJSONValue(value, language)
	return json.Marshal(value)
}

func LocalizeRawJSONMap(values map[string]json.RawMessage, language string) (map[string]json.RawMessage, error) {
	if values == nil {
		return nil, nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for name, raw := range values {
		localized, err := LocalizeRawJSON(raw, language)
		if err != nil {
			return nil, fmt.Errorf("localize component %q: %w", name, err)
		}
		result[name] = localized
	}
	return result, nil
}

func localizeJSONValue(value any, language string) any {
	switch typed := value.(type) {
	case []any:
		for i := range typed {
			typed[i] = localizeJSONValue(typed[i], language)
		}
	case map[string]any:
		for key, child := range typed {
			typed[key] = localizeJSONValue(child, language)
		}
		projectLocalizedField(typed, "title", language)
		projectLocalizedField(typed, "description", language)
	}
	return value
}

func projectLocalizedField(node map[string]any, field, language string) {
	enValue, enExists := node[field+"_en"]
	zhValue, zhExists := node[field+"_zh"]
	if !enExists && !zhExists {
		return
	}
	en, enOK := enValue.(string)
	zh, zhOK := zhValue.(string)
	if enExists && !enOK || zhExists && !zhOK {
		return
	}
	base, baseOK := node[field].(string)
	if _, exists := node[field]; exists && !baseOK {
		return
	}
	delete(node, field+"_en")
	delete(node, field+"_zh")
	selected := en
	if language == "zh" {
		selected = firstNonEmpty(zh, base, en)
	} else {
		selected = firstNonEmpty(en, base, zh)
	}
	if selected == "" {
		delete(node, field)
	} else {
		node[field] = selected
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func localizeDocumentRawJSON(document any, language string) (any, error) {
	localized := document
	switch typed := document.(type) {
	case *APIResponseDocument:
		if typed == nil {
			return typed, nil
		}
		copy := *typed
		var err error
		copy.Responses, err = LocalizeRawJSON(typed.Responses, language)
		if err != nil {
			return nil, err
		}
		copy.Components, err = LocalizeRawJSONMap(typed.Components, language)
		if err != nil {
			return nil, err
		}
		if typed.OutputSchema != nil {
			output := *typed.OutputSchema
			output.Schema, err = LocalizeRawJSON(output.Schema, language)
			if err != nil {
				return nil, err
			}
			output.Components, err = LocalizeRawJSONMap(output.Components, language)
			if err != nil {
				return nil, err
			}
			copy.OutputSchema = &output
		}
		localized = &copy
	}
	raw, err := json.Marshal(localized)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return localizeDocumentValue(value, language), nil
}

func localizeDocumentValue(value any, language string) any {
	switch typed := value.(type) {
	case []any:
		for i := range typed {
			typed[i] = localizeDocumentValue(typed[i], language)
		}
	case map[string]any:
		if selected, ok := localizedTextValue(typed, language); ok {
			return selected
		}
		for key, child := range typed {
			typed[key] = localizeDocumentValue(child, language)
		}
	}
	return value
}

func localizedTextValue(value map[string]any, language string) (string, bool) {
	if len(value) == 0 {
		return "", false
	}
	for key := range value {
		if key != "en" && key != "zh" {
			return "", false
		}
	}
	en, enOK := value["en"].(string)
	zh, zhOK := value["zh"].(string)
	if _, exists := value["en"]; exists && !enOK {
		return "", false
	}
	if _, exists := value["zh"]; exists && !zhOK {
		return "", false
	}
	if language == "zh" {
		return firstNonEmpty(zh, en), true
	}
	return firstNonEmpty(en, zh), true
}
