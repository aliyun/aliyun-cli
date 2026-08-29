package openapi

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

const noHelpSearchMatchesFormat = "No Help entries matched --help-search %q."

const (
	rootDefaultHelpReservedLines    = 25
	productDefaultHelpReservedLines = 15
)

func applyRootHelpOptions(document *machineHelpRootDocument, options helpOptions, aiMode bool) {
	if document == nil {
		return
	}
	sort.SliceStable(document.Commands, func(i, j int) bool {
		return strings.ToLower(document.Commands[i].Name) < strings.ToLower(document.Commands[j].Name)
	})
	sort.SliceStable(document.Products, func(i, j int) bool {
		return strings.ToLower(document.Products[i].Code) < strings.ToLower(document.Products[j].Code)
	})
	type rootProjectionEntry struct {
		kind    string
		command machineHelpCommandSummary
		flag    machineHelpFlagSummary
		product machineHelpProductSummary
	}
	allFlags := append([]machineHelpFlagSummary(nil), document.GlobalFlags...)
	visibleFlags := make([]machineHelpFlagSummary, 0, len(allFlags))
	for _, flag := range allFlags {
		if options.All || options.Search != "" || flag.Visibility == string(RootVisibilityDefault) {
			visibleFlags = append(visibleFlags, flag)
		}
	}
	entries := make([]rootProjectionEntry, 0, len(document.Commands)+len(visibleFlags)+len(document.Products))
	for _, command := range document.Commands {
		entries = append(entries, rootProjectionEntry{kind: "command", command: command})
	}
	for _, flag := range visibleFlags {
		entries = append(entries, rootProjectionEntry{kind: "flag", flag: flag})
	}
	for _, product := range document.Products {
		entries = append(entries, rootProjectionEntry{kind: "product", product: product})
	}
	searched := options.Search != ""
	if searched {
		document.Query = options.Search
		candidates := make([]HelpSearchCandidate, 0, len(entries))
		for _, entry := range entries {
			if entry.kind == "command" {
				candidates = append(candidates, HelpSearchCandidate{
					Kind:          "command",
					Name:          entry.command.Name,
					Aliases:       entry.command.Aliases,
					DescriptionEN: entry.command.Description.EN,
					DescriptionZH: entry.command.Description.ZH,
					Value:         entry,
				})
				continue
			}
			if entry.kind == "flag" {
				aliases := append([]string(nil), entry.flag.Aliases...)
				if entry.flag.Shorthand != "" {
					aliases = append(aliases, entry.flag.Shorthand)
				}
				candidates = append(candidates, HelpSearchCandidate{
					Kind:          "global-flag",
					Name:          entry.flag.Name,
					Aliases:       aliases,
					DescriptionEN: entry.flag.Description.EN,
					DescriptionZH: entry.flag.Description.ZH,
					Value:         entry,
				})
				continue
			}
			product := entry.product
			productAliases := nonEmptyMachineHelpStrings(product.Name.EN, product.Name.ZH)
			candidates = append(candidates, HelpSearchCandidate{
				Kind:          "product",
				Name:          product.Code,
				Aliases:       productAliases,
				DescriptionEN: product.Name.EN,
				DescriptionZH: product.Name.ZH,
				Value:         entry,
			})
		}
		projection := ProjectHelpSearchMatches(SearchHelpCandidates(candidates, options.Search))
		document.Commands = nil
		document.GlobalFlags = nil
		document.Products = nil
		document.Matches = nil
		for _, match := range projection.Matches {
			entry, ok := match.Candidate.Value.(rootProjectionEntry)
			if !ok {
				continue
			}
			if entry.kind == "command" {
				document.Commands = append(document.Commands, entry.command)
				document.Matches = append(document.Matches, machineHelpRootMatch{
					Kind: "command", Name: entry.command.Name, Aliases: entry.command.Aliases,
					Command: strings.Join(entry.command.Path, " ") + " --help", Description: entry.command.Description,
				})
			} else if entry.kind == "flag" {
				document.GlobalFlags = append(document.GlobalFlags, entry.flag)
				aliases := append([]string(nil), entry.flag.Aliases...)
				if entry.flag.Shorthand != "" {
					aliases = append(aliases, entry.flag.Shorthand)
				}
				document.Matches = append(document.Matches, machineHelpRootMatch{
					Kind: "global-flag", Name: entry.flag.Name,
					Aliases: aliases,
					Command: "aliyun --help-search " + strings.TrimLeft(entry.flag.Name, "-"), Description: entry.flag.Description,
				})
			} else {
				document.Products = append(document.Products, entry.product)
				document.Matches = append(document.Matches, machineHelpRootMatch{
					Kind: "product", Name: entry.product.Code,
					Aliases: nonEmptyMachineHelpStrings(entry.product.Name.EN, entry.product.Name.ZH),
					Command: "aliyun " + entry.product.Code + " --help", Description: entry.product.Name,
				})
			}
		}
		document.Result = projection.Result
		document.Listing = nil
		return
	}

	objects := make([]HelpBudgetObject[rootProjectionEntry], 0, len(entries))
	for _, entry := range entries {
		objects = append(objects, HelpBudgetObject[rootProjectionEntry]{Value: entry, LogicalLines: 1})
	}
	projection := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode:          HelpProjectionMode{AIMode: aiMode, JSON: true, All: options.All},
		ReservedLines: rootDefaultHelpReservedLines,
	})
	document.Commands = nil
	document.GlobalFlags = nil
	document.Products = nil
	document.Matches = nil
	for _, entry := range projection.Items {
		if entry.kind == "command" {
			document.Commands = append(document.Commands, entry.command)
		} else if entry.kind == "flag" {
			document.GlobalFlags = append(document.GlobalFlags, entry.flag)
		} else {
			document.Products = append(document.Products, entry.product)
		}
	}
	document.Result = projection.Result
	document.Next = projection.Next
	document.Listing = nil
}

func nonEmptyMachineHelpStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func applyProductHelpOptions(document *machineHelpProductDocument, options helpOptions, aiMode bool) {
	if document == nil {
		return
	}
	sort.SliceStable(document.APIs, func(i, j int) bool {
		left := document.APIs[i].DisplayName
		if left == "" {
			left = document.APIs[i].Name
		}
		right := document.APIs[j].DisplayName
		if right == "" {
			right = document.APIs[j].Name
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})
	searched := options.Search != ""
	if searched {
		document.Query = options.Search
		candidates := make([]HelpSearchCandidate, 0, len(document.APIs))
		for index := range document.APIs {
			api := document.APIs[index]
			candidates = append(candidates, HelpSearchCandidate{
				Kind:          "api",
				Name:          firstNonEmptyMachineHelpString(api.DisplayName, api.Name),
				Aliases:       []string{api.Name, api.CmdName},
				TitleEN:       api.Title.EN,
				TitleZH:       api.Title.ZH,
				DescriptionEN: api.Description.EN,
				DescriptionZH: api.Description.ZH,
				Value:         api,
			})
		}
		projection := ProjectHelpSearchMatches(SearchHelpCandidates(candidates, options.Search))
		document.APIs = helpSearchValues[machineHelpAPISummary](projection.Matches)
		document.Result = projection.Result
		document.Listing = nil
		return
	}

	objects := make([]HelpBudgetObject[machineHelpAPISummary], 0, len(document.APIs))
	for _, api := range document.APIs {
		objects = append(objects, HelpBudgetObject[machineHelpAPISummary]{Value: api, LogicalLines: 1})
	}
	projection := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode:          HelpProjectionMode{AIMode: aiMode, JSON: true, All: options.All},
		ReservedLines: productDefaultHelpReservedLines,
	})
	document.APIs = projection.Items
	document.Result = projection.Result
	document.Next = projection.Next
	document.Listing = nil
	if !options.All && !showProductActionDescriptionsInDefaultHelp {
		for index := range document.APIs {
			document.APIs[index].Title = machineHelpLocalizedText{}
			document.APIs[index].Description = machineHelpLocalizedText{}
		}
	}
}

// applyActionHelpOptions projects an L2 Action summary. Explicit Request
// Sections continue to use applyRequestHelpOptions and remain complete.
func applyActionHelpOptions(document *machineHelpAPIDocument, options helpOptions, aiMode, jsonOutput bool) {
	if document == nil {
		return
	}
	if options.Search != "" {
		applyRequestHelpOptions(document, options, aiMode)
		retainActiveMachineHelpExample(document)
		return
	}
	parameters := activeMachineHelpParameters(document)
	if options.All {
		retainActiveMachineHelpParameterSet(document)
		retainActiveMachineHelpExample(document)
		document.Result = HelpResult{
			Shown: len(parameters) + len(document.GlobalParameters),
			Total: len(parameters) + len(document.GlobalParameters),
		}
		return
	}

	objects := make([]HelpBudgetObject[machineHelpParameter], 0, len(parameters))
	for _, parameter := range parameters {
		objects = append(objects, HelpBudgetObject[machineHelpParameter]{
			Value:        parameter,
			LogicalLines: machineHelpParameterLogicalLines(parameter),
			Required:     parameter.Required,
		})
	}
	projection := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode:           HelpProjectionMode{AIMode: aiMode, JSON: jsonOutput},
		ReservedLines:  8,
		RequiredSafety: true,
	})
	if document.ActiveParameterSet == "kebab" {
		document.ParameterSets.Kebab = projection.Items
		document.ParameterSets.Camel = nil
	} else {
		document.ParameterSets.Camel = projection.Items
		document.ParameterSets.Kebab = nil
	}
	retainActiveMachineHelpExample(document)
	document.GlobalParameters = nil
	document.Result = projection.Result
	document.Next = projection.Next
	// Default shows one compact purpose: title when present, otherwise the
	// description fallback. Structured All/Section retains both fields.
	if document.API.Title.EN != "" || document.API.Title.ZH != "" {
		document.API.Description = machineHelpLocalizedText{}
	}
	apiStyle := strings.ToLower(document.Product.APIStyle)
	if apiStyle == "" {
		// Metadata-plugin products carry the style per API, not per product.
		apiStyle = strings.ToLower(document.API.Operation.APIStyle)
	}
	if apiStyle != "roa" && apiStyle != "rest" && apiStyle != "restful" {
		document.API.Operation.Method = ""
		document.API.Operation.Protocol = ""
		document.API.Operation.URL = ""
		document.API.Operation.IsSSE = false
		document.API.Operation.ReqBodyType = ""
		document.API.Operation.ContentType = ""
		document.API.Operation.HasWildcardPath = false
	}
}

func applyRequestHelpOptions(document *machineHelpAPIDocument, options helpOptions, _ bool) {
	if document == nil {
		return
	}
	document.Listing = nil
	// Explicit Sections stay complete (no budget projection) but still expose
	// only the active command style, matching the default Action Help.
	retainActiveMachineHelpParameterSet(document)
	retainActiveMachineHelpExample(document)
	if options.Search == "" {
		total := len(activeMachineHelpParameters(document)) + len(document.GlobalParameters)
		document.Result = HelpResult{Shown: total, Total: total}
		return
	}
	document.Query = options.Search

	active := activeMachineHelpParameters(document)
	parameterCandidates := make([]HelpSearchCandidate, 0, len(active))
	for index := range active {
		parameter := active[index]
		parameterCandidates = append(parameterCandidates, machineHelpParameterCandidate(parameter, "parameter"))
	}
	globalCandidates := make([]HelpSearchCandidate, 0, len(document.GlobalParameters))
	for index := range document.GlobalParameters {
		parameter := document.GlobalParameters[index]
		globalCandidates = append(globalCandidates, machineHelpParameterCandidate(parameter, "global-parameter"))
	}
	projection := ProjectHelpSearchMatches(SearchHelpParameters(HelpParameterSearchInput{
		ActiveParameterSet: document.ActiveParameterSet,
		ParameterSets: map[string][]HelpSearchCandidate{
			document.ActiveParameterSet: parameterCandidates,
		},
		GlobalParameters: globalCandidates,
	}, options.Search))

	parameters := make([]machineHelpParameter, 0)
	globals := make([]machineHelpParameter, 0)
	for _, match := range projection.Matches {
		value, ok := match.Candidate.Value.(machineHelpParameter)
		if !ok {
			continue
		}
		if match.Candidate.Kind == "global-parameter" {
			globals = append(globals, value)
		} else {
			parameters = append(parameters, value)
		}
	}

	document.ParameterSets = machineHelpParameterSets{}
	if document.ActiveParameterSet == "kebab" {
		document.ParameterSets.Kebab = parameters
	} else {
		document.ParameterSets.Camel = parameters
	}
	document.GlobalParameters = globals
	document.Result = projection.Result
}

func retainActiveMachineHelpParameterSet(document *machineHelpAPIDocument) {
	if document == nil {
		return
	}
	if document.ActiveParameterSet == "kebab" {
		document.ParameterSets.Camel = nil
		return
	}
	document.ParameterSets.Kebab = nil
}

func retainActiveMachineHelpExample(document *machineHelpAPIDocument) {
	if document == nil {
		return
	}
	if document.ActiveParameterSet == "kebab" {
		document.Examples.Camel = ""
		return
	}
	document.Examples.Kebab = ""
}

func applyResponseHelpOptions(document *machineHelpAPIResponseDocument, options helpOptions) error {
	if document == nil || options.Search == "" {
		return nil
	}
	document.Query = options.Search
	// Search projects only matched response structure. The complete unfiltered
	// Section remains available from a separate default/all document.
	document.Responses = nil
	document.Components = nil
	if document.OutputSchema == nil {
		document.Result = HelpResult{}
		document.Notice = fmt.Sprintf(noHelpSearchMatchesFormat, options.Search)
		return nil
	}
	input := helpResponseSchema(document)
	result, err := SearchResponseSchema(input, options.Search)
	if err != nil {
		return err
	}
	if len(result.Paths) == 0 {
		document.OutputSchema = nil
		document.Matches = nil
		document.Result = result.Result
		document.Notice = fmt.Sprintf(noHelpSearchMatchesFormat, options.Search)
		return nil
	}
	document.OutputSchema.Schema = result.Schema
	document.OutputSchema.Components = nil
	if len(result.Components) > 0 {
		document.OutputSchema.Components = &machineHelpComponents{Schemas: result.Components}
	}
	document.Matches = result.Paths
	document.Result = result.Result
	return nil
}

func helpResponseSchema(document *machineHelpAPIResponseDocument) HelpResponseSchema {
	result := HelpResponseSchema{}
	if document == nil || document.OutputSchema == nil {
		return result
	}
	result.Schema = document.OutputSchema.Schema
	if document.OutputSchema.Components != nil {
		result.Components = document.OutputSchema.Components.Schemas
	}
	return result
}

func responseCommandStyle(style string) ResponseCommandStyle {
	if style == "kebab" {
		return ResponseCommandStyleKebab
	}
	return ResponseCommandStylePascal
}

func activeMachineHelpParameters(document *machineHelpAPIDocument) []machineHelpParameter {
	if document == nil {
		return nil
	}
	if document.ActiveParameterSet == "kebab" {
		return document.ParameterSets.Kebab
	}
	return document.ParameterSets.Camel
}

func machineHelpParameterCandidate(parameter machineHelpParameter, kind string) HelpSearchCandidate {
	aliases := make([]string, 0, len(parameter.Options)+8)
	aliases = append(aliases, parameter.RawName)
	aliases = append(aliases, parameter.Options...)
	descriptionsEN := []string{parameter.Help.EN}
	descriptionsZH := []string{parameter.Help.ZH}
	appendMachineHelpParameterSearchDocument(parameter, &descriptionsEN, &descriptionsZH)
	collectMachineHelpShapeSearchText(parameter.Fields, parameter.Element, parameter.Value, &aliases, &descriptionsEN, &descriptionsZH)
	return HelpSearchCandidate{
		Kind:          kind,
		Name:          parameter.Name,
		Aliases:       aliases,
		DescriptionEN: strings.Join(descriptionsEN, "\n"),
		DescriptionZH: strings.Join(descriptionsZH, "\n"),
		Value:         parameter,
	}
}

func collectMachineHelpShapeSearchText(fields []machineHelpParameter, element, value *machineHelpShape, aliases, descriptionsEN, descriptionsZH *[]string) {
	for index := range fields {
		field := fields[index]
		*aliases = append(*aliases, field.Name, field.RawName)
		*aliases = append(*aliases, field.Options...)
		*descriptionsEN = append(*descriptionsEN, field.Help.EN)
		*descriptionsZH = append(*descriptionsZH, field.Help.ZH)
		appendMachineHelpParameterSearchDocument(field, descriptionsEN, descriptionsZH)
		collectMachineHelpShapeSearchText(field.Fields, field.Element, field.Value, aliases, descriptionsEN, descriptionsZH)
	}
	for _, shape := range []*machineHelpShape{element, value} {
		if shape != nil {
			appendMachineHelpShapeSearchDocument(shape, descriptionsEN, descriptionsZH)
			collectMachineHelpShapeSearchText(shape.Fields, shape.Element, shape.Value, aliases, descriptionsEN, descriptionsZH)
		}
	}
}

func appendMachineHelpParameterSearchDocument(parameter machineHelpParameter, descriptionsEN, descriptionsZH *[]string) {
	values := []string{
		parameter.Example,
		strings.Join(parameter.Constraints.Enum, " "),
		parameter.Constraints.Pattern,
		parameter.Constraints.Minimum,
		parameter.Constraints.Maximum,
		parameter.Constraints.MinLength,
		parameter.Constraints.MaxLength,
	}
	*descriptionsEN = append(*descriptionsEN, values...)
	*descriptionsZH = append(*descriptionsZH, values...)
}

func appendMachineHelpShapeSearchDocument(shape *machineHelpShape, descriptionsEN, descriptionsZH *[]string) {
	if shape == nil {
		return
	}
	values := []string{
		strings.Join(shape.Constraints.Enum, " "),
		shape.Constraints.Pattern,
		shape.Constraints.Minimum,
		shape.Constraints.Maximum,
		shape.Constraints.MinLength,
		shape.Constraints.MaxLength,
	}
	*descriptionsEN = append(*descriptionsEN, values...)
	*descriptionsZH = append(*descriptionsZH, values...)
}

func helpSearchValues[T any](matches []HelpSearchMatch) []T {
	values := make([]T, 0, len(matches))
	for _, match := range matches {
		if value, ok := match.Candidate.Value.(T); ok {
			values = append(values, value)
		}
	}
	return values
}

func projectMachineHelpListing(listing *HelpListingMetadata) *machineHelpListing {
	if listing == nil {
		return nil
	}
	return &machineHelpListing{Shown: listing.Shown, Total: listing.Total, Hint: listing.Hint}
}

// validateRecoverySearch runs the same local projection used by real Help.
// It never calls the remote plugin catalog and refuses to advertise host
// Canonical search for a product whose installed plugin owns text Help.
func (c *Commando) validateRecoverySearch(ctx *cli.Context, request RecoverySearchRequest) bool {
	if c == nil || c.library == nil || c.library.helpRepo == nil || strings.TrimSpace(request.Keyword) == "" {
		return false
	}
	c.loadLocalPlugins()
	// Go plugins own their text Help, so their products cannot be searched
	// through the host. Metadata plugins are served by host Machine Help via
	// the engine loader, so search stays available for them.
	if request.Product != "" && c.hasInstalledProductPlugin(request.Product) && !c.installedMetaPluginProduct(request.Product) {
		return false
	}

	service := newMachineHelpService(c.library.helpRepo)
	options := helpOptions{Section: helpSectionRequest, Search: request.Keyword}
	if request.Section == helpSectionResponse {
		options.Section = helpSectionResponse
	}
	switch {
	case request.Product == "":
		if ctx == nil || ctx.Command() == nil {
			return false
		}
		document, err := c.buildRootHelpDocument(ctx.Command())
		if err != nil {
			return false
		}
		applyRootHelpOptions(document, options, false)
		return len(document.Commands) > 0 || len(document.GlobalFlags) > 0 || len(document.Products) > 0

	case request.API == "":
		product, err := service.findProduct(request.Product)
		if err != nil {
			return false
		}
		versions := normalizedVersions(*product)
		style := request.Style
		if style == "pascal" {
			style = "camel"
		}
		selected, err := selectAPIVersion(*product, versions, request.Version, style)
		if err != nil {
			return false
		}
		document, err := service.buildProductForStyle(request.Product, selected, style)
		if err != nil {
			return false
		}
		applyProductHelpOptions(document, options, false)
		return len(document.APIs) > 0

	case options.Section == helpSectionResponse:
		document, err := service.buildAPIResponse(request.Product, request.API, request.Version)
		if err != nil {
			return false
		}
		if err := applyResponseHelpOptions(document, options); err != nil {
			return false
		}
		return len(document.Matches) > 0

	default:
		document, err := service.buildAPI(request.Product, request.API, request.Version)
		if err != nil {
			return false
		}
		if ctx != nil {
			document.GlobalParameters = projectGlobalParameters(ctx.Flags())
		}
		applyRequestHelpOptions(document, options, false)
		return len(activeMachineHelpParameters(document)) > 0 || len(document.GlobalParameters) > 0
	}
}

func renderCanonicalRootText(w io.Writer, document *machineHelpRootDocument, search string) error {
	if document == nil {
		return fmt.Errorf("root Help document is nil")
	}
	version := firstNonEmptyMachineHelpString(document.Version, cli.Version)
	if _, err := fmt.Fprintf(w, "Alibaba Cloud Command Line Interface Version %s\n", version); err != nil {
		return err
	}
	if search == "" && len(document.QuickStart) > 0 {
		if _, err := fmt.Fprintln(w, "\nQuick Start:"); err != nil {
			return err
		}
		for _, command := range document.QuickStart {
			if _, err := fmt.Fprintf(w, "  %s\n", command); err != nil {
				return err
			}
		}
	}
	if len(document.Commands) == 0 && len(document.GlobalFlags) == 0 && len(document.Products) == 0 && len(document.Matches) == 0 && search != "" {
		_, err := fmt.Fprintf(w, noHelpSearchMatchesFormat+"\n", search)
		return err
	}
	if search != "" && len(document.Matches) > 0 {
		if _, err := fmt.Fprintln(w, "Matches:"); err != nil {
			return err
		}
		for _, match := range document.Matches {
			if _, err := fmt.Fprintf(w, "  %-14s %-20s %s\n", match.Kind, match.Name, localizedMachineHelpText(match.Description)); err != nil {
				return err
			}
			if match.Command != "" {
				if _, err := fmt.Fprintf(w, "    %s\n", match.Command); err != nil {
					return err
				}
			}
		}
		return renderHelpProjectionResult(w, "matches", document.Result, document.Next)
	}
	if len(document.Commands) > 0 {
		renderGroup := func(name, label string) error {
			entries := make([]machineHelpCommandSummary, 0)
			for _, command := range document.Commands {
				commandGroup := command.Group
				if commandGroup == "" {
					commandGroup = "core"
				}
				if commandGroup == name {
					entries = append(entries, command)
				}
			}
			if len(entries) == 0 {
				return nil
			}
			if _, err := fmt.Fprintf(w, "\n%s:\n", label); err != nil {
				return err
			}
			for _, command := range entries {
				name := command.Name
				if len(command.Path) > 1 {
					name = strings.Join(command.Path[1:], " ")
				}
				if _, err := fmt.Fprintf(w, "  %-20s %s\n", name, localizedMachineHelpText(command.Description)); err != nil {
					return err
				}
			}
			return nil
		}
		if err := renderGroup("core", "Core Commands"); err != nil {
			return err
		}
		if err := renderGroup("utils", "Utilities"); err != nil {
			return err
		}
	}
	if len(document.GlobalFlags) > 0 {
		if _, err := fmt.Fprintln(w, "\nGlobal Flags:"); err != nil {
			return err
		}
		for _, flag := range document.GlobalFlags {
			name := flag.Name
			if flag.Shorthand != "" {
				name += ", " + flag.Shorthand
			}
			if _, err := fmt.Fprintf(w, "  %-20s %s\n", name, localizedMachineHelpText(flag.Description)); err != nil {
				return err
			}
		}
	}
	if len(document.Commands) > 0 {
		entries := make([]machineHelpCommandSummary, 0)
		for _, command := range document.Commands {
			if command.Group == "extension" {
				entries = append(entries, command)
			}
		}
		if len(entries) > 0 {
			if _, err := fmt.Fprintln(w, "\nExtensions:"); err != nil {
				return err
			}
			for _, command := range entries {
				name := command.Name
				if len(command.Path) > 1 {
					name = strings.Join(command.Path[1:], " ")
				}
				if _, err := fmt.Fprintf(w, "  %-20s %s\n", name, localizedMachineHelpText(command.Description)); err != nil {
					return err
				}
			}
		}
	}
	if len(document.Products) > 0 || search == "" {
		if _, err := fmt.Fprintln(w, "\nProducts:"); err != nil {
			return err
		}
		for _, product := range document.Products {
			name := localizedMachineHelpText(product.Name)
			if _, err := fmt.Fprintf(w, "  %-20s %s\n", product.Code, name); err != nil {
				return err
			}
		}
	}
	if err := renderTextListing(w, "products", document.Listing); err != nil {
		return err
	}
	return renderHelpProjectionResult(w, "entries", document.Result, document.Next)
}

func renderCanonicalProductText(w io.Writer, document *machineHelpProductDocument, search string) error {
	if document == nil {
		return fmt.Errorf("product Help document is nil")
	}
	name := localizedMachineHelpText(document.Product.Name)
	if _, err := fmt.Fprintf(w, "\nProduct: %s (%s)\nVersion: %s\n", document.Product.Code, name, document.Product.SelectedVersion); err != nil {
		return err
	}
	if provider := machineHelpPluginProvider(document.Product); provider != "" {
		if _, err := fmt.Fprintf(w, "Provided by plugin: %s\n", provider); err != nil {
			return err
		}
	}
	if len(document.APIs) == 0 && search != "" {
		_, err := fmt.Fprintf(w, "\n"+noHelpSearchMatchesFormat+"\n", search)
		return err
	}
	if _, err := fmt.Fprintln(w, "\nAvailable API List:"); err != nil {
		return err
	}
	for _, api := range document.APIs {
		name := api.DisplayName
		if name == "" {
			name = firstNonEmptyMachineHelpString(api.CmdName, api.Name)
		}
		description := localizedMachineHelpPurpose(api.Title, api.Description)
		fullDescription := localizedMachineHelpText(api.Description)
		if description != "" && fullDescription != "" && description != fullDescription {
			description += " — " + fullDescription
		}
		if _, err := fmt.Fprintf(w, "  %-30s %s\n", name, description); err != nil {
			return err
		}
	}
	if err := renderTextListing(w, "APIs", document.Listing); err != nil {
		return err
	}
	return renderHelpProjectionResult(w, "APIs", document.Result, document.Next)
}

func renderCanonicalRequestSearchText(w io.Writer, document *machineHelpAPIDocument, search string) error {
	if document == nil {
		return fmt.Errorf("request Help document is nil")
	}
	if provider := machineHelpPluginProvider(document.Product); provider != "" {
		if _, err := fmt.Fprintf(w, "\nProvided by plugin: %s\n", provider); err != nil {
			return err
		}
	}
	parameters := activeMachineHelpParameters(document)
	if len(parameters) == 0 && len(document.GlobalParameters) == 0 {
		_, err := fmt.Fprintf(w, noHelpSearchMatchesFormat+"\n", search)
		return err
	}
	if len(parameters) > 0 {
		if _, err := fmt.Fprintln(w, "\nParameters:"); err != nil {
			return err
		}
		if err := renderMachineHelpParameters(w, parameters); err != nil {
			return err
		}
	}
	if len(document.GlobalParameters) > 0 {
		if _, err := fmt.Fprintln(w, "\nGlobal Parameters:"); err != nil {
			return err
		}
		if err := renderMachineHelpParameters(w, document.GlobalParameters); err != nil {
			return err
		}
	}
	return nil
}

func renderCanonicalRequestText(w io.Writer, document *machineHelpAPIDocument) error {
	if document == nil {
		return fmt.Errorf("request Help document is nil")
	}
	description := localizedMachineHelpPurpose(document.API.Title, document.API.Description)
	fullDescription := localizedMachineHelpText(document.API.Description)
	details := ""
	if fullDescription != "" && fullDescription != description {
		details = "\nDetails: " + fullDescription
	}
	if _, err := fmt.Fprintf(w,
		"Alibaba Cloud Command Line Interface Version %s\n",
		cli.Version,
	); err != nil {
		return err
	}
	if provider := machineHelpPluginProvider(document.Product); provider != "" {
		if _, err := fmt.Fprintf(w, "\nProvided by plugin: %s\n", provider); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w,
		"\nDescription: %s%s\n\nAPI Version: %s\n\nUsage:\n  %s [parameters]\n",
		description,
		details,
		document.API.Operation.APIVersion,
		strings.Join(document.Target.Path, " "),
	); err != nil {
		return err
	}

	parameters := activeMachineHelpParameters(document)
	if len(parameters) > 0 {
		if _, err := fmt.Fprintln(w, "\nParameters:"); err != nil {
			return err
		}
		if err := renderMachineHelpParameters(w, parameters); err != nil {
			return err
		}
	}
	if len(document.GlobalParameters) > 0 {
		if _, err := fmt.Fprintln(w, "\nGlobal Parameters:"); err != nil {
			return err
		}
		if err := renderMachineHelpParameters(w, document.GlobalParameters); err != nil {
			return err
		}
	}
	if err := renderTextListing(w, "parameters", document.Listing); err != nil {
		return err
	}
	if err := renderHelpProjectionResult(w, "parameters", document.Result, document.Next); err != nil {
		return err
	}

	example := document.Examples.Camel
	if document.ActiveParameterSet == "kebab" {
		example = document.Examples.Kebab
	}
	if strings.TrimSpace(example) != "" {
		if _, err := fmt.Fprintf(w, "\nExample:\n  %s\n", example); err != nil {
			return err
		}
	}
	return renderRequestQueryExampleText(w, document.ResponseQuery)
}

// projectOriginalRequestHelpText keeps the established runtime/legacy Help
// renderer intact and changes only the API parameter blocks. Global parameters,
// examples, wrapping, labels, and every other section remain byte-for-byte as
// rendered by the original provider.
func projectOriginalRequestHelpText(text string, options helpOptions, aiMode bool) (string, *HelpListingMetadata) {
	lines := strings.SplitAfter(text, "\n")
	parametersLine := -1
	globalParametersLine := -1
	for index, line := range lines {
		switch strings.TrimSpace(line) {
		case "Parameters:", "参数:":
			parametersLine = index
		case "Global Parameters:", "全局参数:":
			if parametersLine >= 0 {
				globalParametersLine = index
			}
		}
	}
	if parametersLine < 0 || globalParametersLine <= parametersLine {
		return text, nil
	}

	type parameterBlock struct {
		text string
	}
	blocks := make([]parameterBlock, 0)
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		blocks = append(blocks, parameterBlock{text: current.String()})
		current.Reset()
	}
	for _, line := range lines[parametersLine+1 : globalParametersLine] {
		if strings.HasPrefix(line, "  --") {
			flush()
		}
		if len(blocks) == 0 && current.Len() == 0 && strings.TrimSpace(line) == "" {
			continue
		}
		current.WriteString(line)
	}
	flush()
	if len(blocks) == 0 {
		return text, nil
	}

	selected := blocks
	if options.Search != "" {
		query := newHelpSearchText(options.Search)
		selected = make([]parameterBlock, 0, len(blocks))
		for _, block := range blocks {
			if helpTextContains(stripHelpANSI(block.text), query) {
				selected = append(selected, block)
			}
		}

	}

	var listing *HelpListingMetadata
	if aiMode && options.Search == "" && !options.All && len(selected) > helpListingLimit {
		listing = &HelpListingMetadata{Shown: helpListingLimit, Total: len(selected), Hint: helpListingHint}
		selected = selected[:helpListingLimit]
	}
	if options.Search == "" && listing == nil {
		return text, nil
	}

	var projected strings.Builder
	for _, line := range lines[:parametersLine+1] {
		projected.WriteString(line)
	}
	if len(selected) == 0 {
		fmt.Fprintf(&projected, noHelpSearchMatchesFormat+"\n", options.Search)
	} else {
		for _, block := range selected {
			projected.WriteString(strings.TrimRight(block.text, "\n"))
			projected.WriteByte('\n')
		}
	}
	if listing != nil {
		fmt.Fprintf(&projected, "\nShowing %d of %d parameters.\n%s\n", listing.Shown, listing.Total, listing.Hint)
	}
	projected.WriteByte('\n')
	for _, line := range lines[globalParametersLine:] {
		projected.WriteString(line)
	}
	return projected.String(), listing
}

func stripHelpANSI(value string) string {
	for {
		start := strings.Index(value, "\x1b[")
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start:], 'm')
		if end < 0 {
			return value
		}
		value = value[:start] + value[start+end+1:]
	}
}

const machineHelpParameterNameWidth = 30

// machineHelpPluginProvider renders "name (version)" for a plugin-served
// product, or "" for baseline-served products.
func machineHelpPluginProvider(product machineHelpProduct) string {
	if product.Plugin == "" {
		return ""
	}
	if product.PluginVersion != "" {
		return product.Plugin + " (" + product.PluginVersion + ")"
	}
	return product.Plugin
}

func renderMachineHelpParameters(w io.Writer, parameters []machineHelpParameter) error {
	// Continuation lines align under the type column, mirroring the engine's
	// parameter Help layout, and leave room for readable wrapped text.
	indent := strings.Repeat(" ", 2+machineHelpParameterNameWidth+1)
	wrapWidth := machineHelpMaxLineLength() - len(indent)
	for _, parameter := range parameters {
		name := parameter.Name
		if len(parameter.Options) > 0 {
			name = strings.Join(parameter.Options, ", ")
		}
		requiredLabel := "optional"
		if parameter.Required {
			requiredLabel = "required"
		}
		prefix := fmt.Sprintf("  %-*s %s (%s)", machineHelpParameterNameWidth, name, parameter.Type, requiredLabel)
		help := localizedMachineHelpText(parameter.Help)
		lines := wrapMachineHelpText(help, wrapWidth)
		if len(lines) == 1 && len([]rune(prefix))+2+len([]rune(lines[0])) <= machineHelpMaxLineLength() {
			if _, err := fmt.Fprintf(w, "%s  %s\n", prefix, lines[0]); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\n", prefix); err != nil {
			return err
		}
		for _, line := range lines {
			if _, err := fmt.Fprintf(w, "%s%s\n", indent, line); err != nil {
				return err
			}
		}
	}
	return nil
}

func machineHelpMaxLineLength() int {
	if value := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CLI_MAX_LINE_LENGTH")); value != "" {
		if length, err := strconv.Atoi(value); err == nil && length > 0 {
			return length
		}
	}
	return 80
}

// wrapMachineHelpText reflows parameter Help text into fixed-width lines:
// embedded blank lines are collapsed and long lines wrap at spaces or CJK
// punctuation so continuation lines stay inside the Help column.
func wrapMachineHelpText(text string, width int) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if width < 12 {
		width = 12
	}
	lines := make([]string, 0, 2)
	for _, paragraph := range strings.Split(text, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		lines = append(lines, wrapMachineHelpLine(paragraph, width)...)
	}
	return lines
}

func wrapMachineHelpLine(text string, width int) []string {
	runes := []rune(text)
	if len(runes) <= width {
		return []string{text}
	}
	lines := make([]string, 0, len(runes)/width+1)
	for start := 0; start < len(runes); {
		end := start + width
		if end >= len(runes) {
			lines = append(lines, string(runes[start:]))
			break
		}
		breakPoint := wrapBreakPoint(runes, start, end)
		lines = append(lines, strings.TrimSpace(string(runes[start:breakPoint])))
		start = breakPoint
		for start < len(runes) && runes[start] == ' ' {
			start++
		}
	}
	return lines
}

// wrapBreakPoint finds a break position in the second half of the window so
// wrapped lines keep a coherent chunk on each side. Spaces, ASCII and CJK
// punctuation, and slashes (URLs and paths carry no spaces) all qualify; the
// latest qualifying rune wins.
func wrapBreakPoint(runes []rune, start, end int) int {
	for i := end - 1; i > start+(end-start)/2; i-- {
		switch runes[i] {
		case ' ', ',', '.', ';', '/', '，', '。', '、', '：':
			return i + 1
		}
	}
	return end
}

func renderTextListing(w io.Writer, noun string, listing *machineHelpListing) error {
	if listing == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\nShowing %d of %d %s.\n%s\n", listing.Shown, listing.Total, noun, listing.Hint); err != nil {
		return err
	}
	return nil
}

func renderHelpProjectionResult(w io.Writer, noun string, result HelpResult, next *HelpNext) error {
	if !result.Truncated {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\n...\nShowing %d of %d %s.\n", result.Shown, result.Total, noun); err != nil {
		return err
	}
	if next != nil {
		if next.ShowAll != "" {
			if _, err := fmt.Fprintf(w, "Show all: %s\n", next.ShowAll); err != nil {
				return err
			}
		}
		if next.Search != "" {
			if _, err := fmt.Fprintf(w, "Search: %s\n", next.Search); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := fmt.Fprintln(w, "Use a more specific --help-search query.")
	return err
}

func renderRequestQueryExampleText(w io.Writer, example *machineHelpQueryExample) error {
	if example == nil {
		return nil
	}
	_, err := fmt.Fprintf(w,
		"\nResponse query example (%s):\n"+
			"This response contains a complex array. Inspect its structure with the response section, then use --cli-query to return only that array:\n"+
			"  %s\n  %s\n",
		example.Path, example.SchemaCommand, example.QueryCommand)
	return err
}

func renderAIModeEnableHelpHint(w io.Writer) error {
	_, err := fmt.Fprintf(w, "\n%s\n", cli.AIModeEnableTextHint)
	return err
}

func (c *Commando) finishCanonicalTextHelp(ctx *cli.Context, aiMode bool) error {
	if aiMode {
		// These specialized Canonical renderers bypass the historical text
		// route, so explicitly preserve its configured AI-mode disable hint.
		c.printAiModeHelpHint(ctx)
		return nil
	}
	return renderAIModeEnableHelpHint(ctx.Stdout())
}

func localizedMachineHelpText(text machineHelpLocalizedText) string {
	if i18n.GetLanguage() == "zh" && text.ZH != "" {
		return text.ZH
	}
	if text.EN != "" {
		return text.EN
	}
	return text.ZH
}
