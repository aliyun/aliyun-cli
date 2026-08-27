package openapi

import (
	"fmt"
	"io"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/i18n"
)

const noHelpSearchMatchesFormat = "No Help entries matched --cli-search %q."

func applyRootHelpOptions(document *machineHelpRootDocument, options helpOptions, aiMode bool) {
	if document == nil {
		return
	}
	searched := options.Search != ""
	if searched {
		candidates := make([]HelpSearchCandidate, 0, len(document.Products))
		for index := range document.Products {
			product := document.Products[index]
			candidates = append(candidates, HelpSearchCandidate{
				Kind:          "product",
				Name:          product.Code,
				Aliases:       []string{product.Name.EN, product.Name.ZH},
				DescriptionEN: product.Name.EN,
				DescriptionZH: product.Name.ZH,
				Value:         product,
			})
		}
		matches := SearchHelpCandidates(candidates, options.Search)
		document.Products = helpSearchValues[machineHelpProductSummary](matches)
		// Root Search is a product search. Built-in commands are deliberately
		// omitted so unrelated command entries do not dilute the result.
		document.Commands = nil
	}

	products, listing := ProjectHelpListing(document.Products, HelpListingOptions{
		Target:   HelpListingRootProducts,
		AIMode:   aiMode,
		Searched: searched,
		All:      options.All,
	})
	document.Products = products
	document.Listing = projectMachineHelpListing(listing)
}

func applyProductHelpOptions(document *machineHelpProductDocument, options helpOptions, aiMode bool) {
	if document == nil {
		return
	}
	searched := options.Search != ""
	if searched {
		candidates := make([]HelpSearchCandidate, 0, len(document.APIs))
		for index := range document.APIs {
			api := document.APIs[index]
			candidates = append(candidates, HelpSearchCandidate{
				Kind:          "api",
				Name:          api.Name,
				Aliases:       []string{api.CmdName},
				DescriptionEN: api.Description.EN,
				DescriptionZH: api.Description.ZH,
				Value:         api,
			})
		}
		matches := SearchHelpCandidates(candidates, options.Search)
		document.APIs = helpSearchValues[machineHelpAPISummary](matches)
	}

	apis, listing := ProjectHelpListing(document.APIs, HelpListingOptions{
		Target:   HelpListingProductAPIs,
		AIMode:   aiMode,
		Searched: searched,
		All:      options.All,
	})
	document.APIs = apis
	document.Listing = projectMachineHelpListing(listing)
}

func applyRequestHelpOptions(document *machineHelpAPIDocument, options helpOptions) {
	if document == nil || options.Search == "" {
		return
	}

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
	matches := SearchHelpParameters(HelpParameterSearchInput{
		ActiveParameterSet: document.ActiveParameterSet,
		ParameterSets: map[string][]HelpSearchCandidate{
			document.ActiveParameterSet: parameterCandidates,
		},
		GlobalParameters: globalCandidates,
	}, options.Search)

	parameters := make([]machineHelpParameter, 0)
	globals := make([]machineHelpParameter, 0)
	for _, match := range matches {
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
}

func applyResponseHelpOptions(document *machineHelpAPIResponseDocument, options helpOptions) error {
	if document == nil || document.OutputSchema == nil || options.Search == "" {
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
		document.Notice = fmt.Sprintf(noHelpSearchMatchesFormat, options.Search)
		return nil
	}
	document.OutputSchema.Schema = result.Schema
	document.OutputSchema.Components = nil
	if len(result.Components) > 0 {
		document.OutputSchema.Components = &machineHelpComponents{Schemas: result.Components}
	}
	document.Matches = result.Paths
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
		collectMachineHelpShapeSearchText(field.Fields, field.Element, field.Value, aliases, descriptionsEN, descriptionsZH)
	}
	for _, shape := range []*machineHelpShape{element, value} {
		if shape != nil {
			collectMachineHelpShapeSearchText(shape.Fields, shape.Element, shape.Value, aliases, descriptionsEN, descriptionsZH)
		}
	}
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

func renderCanonicalRootText(w io.Writer, document *machineHelpRootDocument, search string) error {
	if document == nil {
		return fmt.Errorf("root Help document is nil")
	}
	if len(document.Products) == 0 && search != "" {
		_, err := fmt.Fprintf(w, noHelpSearchMatchesFormat+"\n", search)
		return err
	}
	if len(document.Commands) > 0 {
		if _, err := fmt.Fprintln(w, "\nCommands:"); err != nil {
			return err
		}
		for _, command := range document.Commands {
			if _, err := fmt.Fprintf(w, "  %-20s %s\n", command.Name, localizedMachineHelpText(command.Description)); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w, "\nProducts:"); err != nil {
		return err
	}
	for _, product := range document.Products {
		name := localizedMachineHelpText(product.Name)
		if _, err := fmt.Fprintf(w, "  %-20s %s\n", product.Code, name); err != nil {
			return err
		}
	}
	return renderTextListing(w, "products", document.Listing)
}

func renderCanonicalProductText(w io.Writer, document *machineHelpProductDocument, search string) error {
	if document == nil {
		return fmt.Errorf("product Help document is nil")
	}
	name := localizedMachineHelpText(document.Product.Name)
	if _, err := fmt.Fprintf(w, "\nProduct: %s (%s)\nVersion: %s\n", document.Product.Code, name, document.Product.SelectedVersion); err != nil {
		return err
	}
	if len(document.APIs) == 0 && search != "" {
		_, err := fmt.Fprintf(w, "\n"+noHelpSearchMatchesFormat+"\n", search)
		return err
	}
	if _, err := fmt.Fprintln(w, "\nAvailable API List:"); err != nil {
		return err
	}
	for _, api := range document.APIs {
		name := api.CmdName
		if name == "" {
			name = api.Name
		}
		description := localizedMachineHelpText(api.Description)
		if _, err := fmt.Fprintf(w, "  %-30s %s\n", name, description); err != nil {
			return err
		}
	}
	return renderTextListing(w, "APIs", document.Listing)
}

func renderCanonicalRequestSearchText(w io.Writer, document *machineHelpAPIDocument, search string) error {
	if document == nil {
		return fmt.Errorf("request Help document is nil")
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

func renderMachineHelpParameters(w io.Writer, parameters []machineHelpParameter) error {
	for _, parameter := range parameters {
		name := parameter.Name
		if len(parameter.Options) > 0 {
			name = strings.Join(parameter.Options, ", ")
		}
		requiredLabel := "optional"
		if parameter.Required {
			requiredLabel = "required"
		}
		help := localizedMachineHelpText(parameter.Help)
		if _, err := fmt.Fprintf(w, "  %-30s %s (%s)  %s\n", name, parameter.Type, requiredLabel, help); err != nil {
			return err
		}
	}
	return nil
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

func localizedMachineHelpText(text machineHelpLocalizedText) string {
	if i18n.GetLanguage() == "zh" && text.ZH != "" {
		return text.ZH
	}
	if text.EN != "" {
		return text.EN
	}
	return text.ZH
}
