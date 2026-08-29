package openapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/openapi/runtimehost"
)

var (
	goPluginHelpDispatch   = plugin.ExecutePluginRaw
	metaPluginHelpDispatch = runtimehost.DispatchPluginHelp
)

// beforeParseHelpRoute owns the two cases that the generic flag parser cannot
// safely handle: installed-plugin Help (the plugin must see the original
// invocation before host policy is applied) and an unassigned L3 parameter
// immediately followed by a Help operation.
func (c *Commando) beforeParseHelpRoute(ctx *cli.Context, args []string) (bool, error) {
	if removed := removedHostHelpOption(args); removed != "" {
		product, _ := rawHelpProduct(args)
		if product != "" {
			c.loadLocalPlugins()
			if _, _, installed := plugin.FindInstalledPluginInManifest(c.localManifest, product); installed {
				if rawHelpRequested(args) {
					return c.delegateInstalledPluginHelp(ctx, args)
				}
				return false, nil
			}
		}
		return true, cli.NewInvalidFlagError(removed, ctx)
	}
	requested := rawHelpRequested(args)
	positionals := rawHelpPositionals(args)
	if len(positionals) == 0 {
		if unknown := unknownRootOption(ctx, args); unknown != "" {
			return true, cli.NewInvalidFlagError(unknown, ctx)
		}
	}
	if len(positionals) > 0 && positionals[0] != "utils" && ctx != nil && ctx.Command() != nil && ctx.Command().GetSubCommand(positionals[0]) != nil {
		return false, nil
	}
	implicitProductHelp := !requested && len(positionals) == 1
	if !requested && !implicitProductHelp {
		return false, nil
	}
	if delegated, err := c.delegateInstalledPluginHelp(ctx, args); delegated {
		return true, err
	}
	if len(positionals) > 0 && positionals[0] == "utils" {
		opts, err := cli.ParseHelpOptions(args)
		if err != nil {
			return true, err
		}
		if opts.SectionExplicit {
			return true, &InvalidOptionCombinationError{
				Options: []string{"--" + cli.CLISectionFlagName},
				Err:     fmt.Errorf("--%s is only valid for OpenAPI request/response Help", cli.CLISectionFlagName),
			}
		}
		return true, c.renderUtilityHelp(ctx, positionals, opts, c.applyEffectiveAIModeForArgs(ctx, args))
	}
	if !requested {
		return false, nil
	}

	opts, err := cli.ParseHelpOptions(args)
	if err != nil {
		return true, err
	}
	parameter, ambiguous := rawParameterHelpTarget(args)
	if ambiguous {
		return true, &InvalidOptionCombinationError{
			Options: []string{"parameter Help"},
			Err:     fmt.Errorf("parameter Help target is ambiguous; specify exactly one unassigned parameter flag"),
		}
	}
	if parameter == "" {
		return false, nil
	}

	targetArgs := rawHelpPositionals(args)
	if len(targetArgs) != 2 {
		return true, fmt.Errorf("parameter Help requires a product and an API")
	}
	aiMode := c.applyEffectiveAIModeForArgs(ctx, args)
	target := HelpTarget{
		Level:        HelpLevelParameter,
		Product:      targetArgs[0],
		Action:       targetArgs[1],
		Parameter:    parameter,
		CommandStyle: commandStyleForAction(targetArgs[1]),
		Operation:    opts.Operation,
		SearchQuery:  opts.SearchQuery,
		Output:       opts.Output,
		Provider:     HelpProviderHost,
	}
	target.VersionFlag, target.Version = rawExplicitVersion(args)
	return true, c.renderHostHelpTarget(ctx, target, aiMode)
}

func unknownRootOption(ctx *cli.Context, args []string) string {
	if ctx == nil || ctx.Flags() == nil {
		return ""
	}
	for _, arg := range args {
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.HasPrefix(arg, "--help=") {
			return arg
		}
		name := strings.SplitN(arg, "=", 2)[0]
		if strings.HasPrefix(name, "--") {
			if ctx.Flags().Get(strings.TrimPrefix(name, "--")) == nil {
				return name
			}
			continue
		}
		if len([]rune(name)) == 2 {
			runes := []rune(name)
			if ctx.Flags().GetByShorthand(runes[1]) == nil {
				return name
			}
		}
	}
	return ""
}

func removedHostHelpOption(args []string) string {
	for index, arg := range args {
		name := strings.SplitN(arg, "=", 2)[0]
		if strings.HasPrefix(arg, "--help=") {
			return arg
		}
		if name == "--cli-all" || name == "--cli-search" || name == "--help-json" {
			return name
		}
		if index > 0 && args[0] == "help" && name == "--format" {
			return name
		}
	}
	return ""
}

func rawHelpRequested(args []string) bool {
	if len(args) > 0 && args[0] == "help" {
		return true
	}
	for _, arg := range args {
		name := strings.SplitN(arg, "=", 2)[0]
		switch name {
		case "--help", "-h", "--" + cli.HelpAllFlagName, "--" + cli.HelpSearchFlagName:
			return true
		}
	}
	return false
}

func (c *Commando) delegateInstalledPluginHelp(ctx *cli.Context, args []string) (bool, error) {
	product, index := rawHelpProduct(args)
	if product == "" {
		return false, nil
	}
	// Built-in Core, Utils and Extension commands keep their established
	// command-tree ownership. Plugin-first applies only to product targets.
	if ctx != nil && ctx.Command() != nil && ctx.Command().GetSubCommand(product) != nil {
		return false, nil
	}
	c.loadLocalPlugins()
	if c.localManifestErr != nil {
		return true, fmt.Errorf("failed to check plugin status: %w", c.localManifestErr)
	}
	pluginName, local, ok := plugin.FindInstalledPluginInManifest(c.localManifest, product)
	if !ok || local == nil {
		return false, nil
	}
	if err := plugin.ValidateLocalPluginCliVersion(pluginName, local); err != nil {
		return true, &externalPluginError{err: err}
	}

	pluginArgs := append([]string(nil), args[index:]...)
	if (len(pluginArgs) == 1 || (len(args) > 0 && args[0] == "help")) && !containsPluginHelpOperation(pluginArgs) {
		pluginArgs = append(pluginArgs, "--help")
	}
	c.setLangEnv(ctx)
	if local.IsMeta() {
		// Metadata plugins are served by the host's Machine Help (JSON,
		// sections, search) through the engine loader, so a hot-updated
		// plugin renders exactly like the bundled kebab experience.
		// The version contract was already validated above.
		return false, nil
	}
	ok, err := goPluginHelpDispatch(product, pluginArgs, ctx)
	if err != nil {
		return true, &externalPluginError{err: err}
	}
	if !ok {
		return true, &externalPluginError{err: fmt.Errorf("plugin %s not found", pluginName)}
	}
	return true, nil
}

func containsPluginHelpOperation(args []string) bool {
	for _, arg := range args {
		name := strings.SplitN(arg, "=", 2)[0]
		if name == "--help" || name == "-h" || name == "--"+cli.HelpAllFlagName || name == "--"+cli.HelpSearchFlagName {
			return true
		}
	}
	return false
}

func rawHelpProduct(args []string) (string, int) {
	positionals := rawHelpPositionalsWithIndexes(args)
	if len(positionals) == 0 {
		return "", -1
	}
	return positionals[0].value, positionals[0].index
}

type indexedHelpToken struct {
	value string
	index int
}

func rawHelpPositionals(args []string) []string {
	indexed := rawHelpPositionalsWithIndexes(args)
	result := make([]string, 0, len(indexed))
	for _, token := range indexed {
		result = append(result, token.value)
	}
	return result
}

func rawHelpPositionalsWithIndexes(args []string) []indexedHelpToken {
	result := make([]indexedHelpToken, 0, 3)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if i == 0 && arg == "help" {
			continue
		}
		name, hasValue := splitLongHelpOption(arg)
		switch name {
		case "help", "h", cli.HelpAllFlagName:
			continue
		case cli.HelpSearchFlagName, cli.CLIOutputFlagName, cli.CLISectionFlagName, "version", "api-version":
			if !hasValue && i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			// Unknown/business/global flags and their values are not target path
			// components. A value is consumed only before product/action have
			// both been resolved; after that it belongs to a request parameter.
			if !hasValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && len(result) < 2 {
				i++
			}
			continue
		}
		result = append(result, indexedHelpToken{value: arg, index: i})
		if len(result) == 2 {
			// Remaining non-options are request values, not deeper host Help
			// targets. Installed plugins are delegated before this scanner.
			continue
		}
	}
	if len(result) > 2 {
		return result[:2]
	}
	return result
}

func splitLongHelpOption(arg string) (name string, hasValue bool) {
	trimmed := strings.TrimLeft(arg, "-")
	if cut := strings.IndexAny(trimmed, "=:"); cut >= 0 {
		return trimmed[:cut], true
	}
	return trimmed, false
}

func rawParameterHelpTarget(args []string) (string, bool) {
	positionals := rawHelpPositionals(args)
	if len(positionals) < 2 {
		return "", false
	}
	candidates := make([]string, 0, 1)
	for i, arg := range args {
		if !strings.HasPrefix(arg, "--") || strings.ContainsAny(arg, "=:") {
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		switch name {
		case "help", cli.HelpAllFlagName, cli.HelpSearchFlagName, cli.CLIOutputFlagName,
			cli.CLISectionFlagName, "version", "api-version", "cli-ai-mode", "no-cli-ai-mode":
			continue
		}
		// A parameter is assigned only when its next token is a value. If the
		// next token starts another long option, the current parameter is also
		// unassigned; retaining every such candidate lets us reject ambiguous
		// L3 invocations instead of silently choosing the final flag.
		if i+1 < len(args) && !isHelpControlToken(args[i+1]) && !strings.HasPrefix(args[i+1], "--") {
			continue
		}
		candidates = append(candidates, "--"+name)
	}
	if len(candidates) > 1 {
		return "", true
	}
	if len(candidates) == 1 {
		return candidates[0], false
	}
	return "", false
}

func isHelpControlToken(arg string) bool {
	name, _ := splitLongHelpOption(arg)
	switch name {
	case "help", "h", cli.HelpAllFlagName, cli.HelpSearchFlagName, cli.CLIOutputFlagName, cli.CLISectionFlagName:
		return true
	}
	return false
}

func rawExplicitVersion(args []string) (APIVersionFlag, string) {
	for i, arg := range args {
		for _, name := range []string{"version", "api-version"} {
			prefix := "--" + name
			if arg == prefix && i+1 < len(args) {
				return APIVersionFlag(name), args[i+1]
			}
			if strings.HasPrefix(arg, prefix+"=") {
				return APIVersionFlag(name), strings.TrimPrefix(arg, prefix+"=")
			}
		}
	}
	return "", ""
}

func commandStyleForAction(action string) CommandStyle {
	if strings.ToLower(action) == action {
		return CommandStyleKebab
	}
	return CommandStyleCamel
}

func (c *Commando) resolveParsedHelpTarget(ctx *cli.Context, args []string) (HelpTarget, bool, error) {
	opts, err := parseHelpOptions(ctx, args)
	if err != nil {
		return HelpTarget{}, false, err
	}
	if opts.SectionExplicit && opts.All {
		return HelpTarget{}, false, &InvalidOptionCombinationError{
			Options: []string{"--" + CliHelpSectionFlagName, "--" + cli.HelpAllFlagName},
			Err:     fmt.Errorf("Help sections do not support --%s", cli.HelpAllFlagName),
		}
	}
	original := ctx.InvocationArgs()
	prefix := len(original) > 0 && original[0] == "help"
	// Unit/library callers may invoke Help directly without going through
	// Command.Execute. Preserve the public prefix-only Section contract while
	// allowing those callers to supply an already parsed section.
	if len(original) == 0 && opts.SectionExplicit {
		prefix = true
	}
	target := HelpTarget{
		Level:     HelpLevelRoot,
		Operation: HelpOperationDefault,
		Section:   HelpSectionRequest,
		Output:    HelpOutput(opts.Output),
		Provider:  HelpProviderHost,
	}
	if opts.All {
		target.Operation = HelpOperationAll
	} else if opts.Search != "" {
		target.Operation = HelpOperationSearch
		target.SearchQuery = opts.Search
	}
	if len(args) >= 1 {
		target.Level = HelpLevelProduct
		target.Product = args[0]
		target.CommandStyle = CommandStyleCamel
		if productHelpEnvEnabled(baselineProductHelpEnv) || c.installedMetaPluginProduct(args[0]) {
			target.CommandStyle = CommandStyleKebab
		}
	}
	if len(args) >= 2 {
		target.Level = HelpLevelAction
		target.Action = args[1]
		target.CommandStyle = commandStyleForAction(args[1])
	}
	if len(args) > 2 {
		return HelpTarget{}, false, fmt.Errorf("too many Help target arguments: %d", len(args))
	}
	if opts.SectionExplicit && !prefix {
		return HelpTarget{}, false, &InvalidOptionCombinationError{
			Options: []string{"--" + CliHelpSectionFlagName},
			Err:     fmt.Errorf("--%s is only valid with `aliyun help <product> <API>`", CliHelpSectionFlagName),
		}
	}
	if prefix && len(args) == 2 {
		target.SectionExplicit = true
		target.Section = HelpSection(opts.Section)
	} else if opts.SectionExplicit {
		target.SectionExplicit = true
		target.Section = HelpSection(opts.Section)
	}
	target.VersionFlag, target.Version = rawExplicitVersion(original)
	if target.Version == "" {
		target.Version = requestedMachineHelpVersion(ctx)
		if target.Version != "" {
			target.VersionFlag = VersionFlagLegacy
		}
	}
	return target, legacyAIModeEnabled(ctx), target.Validate()
}

func (c *Commando) renderHostHelpTarget(ctx *cli.Context, target HelpTarget, aiMode bool) error {
	if c.library == nil || c.library.helpRepo == nil {
		return fmt.Errorf("canonical Help is unavailable: metadata repository is unavailable")
	}
	if err := target.Validate(); err != nil {
		return err
	}
	service := newMachineHelpService(c.library.helpRepo)
	jsonOutput := aiMode || target.Output == HelpOutputJSON
	opts := helpOptions{
		Section:         string(target.Section),
		SectionExplicit: target.SectionExplicit,
		Search:          target.SearchQuery,
		All:             target.Operation == HelpOperationAll,
		Output:          cli.HelpOutput(target.Output),
	}

	var document any
	var err error
	switch target.Level {
	case HelpLevelRoot:
		document, err = c.buildRootHelpDocument(ctx.Command())
		if err == nil {
			applyRootHelpOptions(document.(*machineHelpRootDocument), opts, aiMode)
			setRootHelpNext(document.(*machineHelpRootDocument), target, aiMode)
		}
	case HelpLevelProduct:
		document, err = service.buildProductForStyle(target.Product, target.Version, string(target.CommandStyle))
		if err == nil {
			applyProductHelpOptions(document.(*machineHelpProductDocument), opts, aiMode)
			setProductHelpNext(document.(*machineHelpProductDocument), target, aiMode)
		}
	case HelpLevelAction:
		if target.SectionExplicit && target.Section == HelpSectionResponse {
			document, err = service.buildAPIResponse(target.Product, target.Action, target.Version)
			if err == nil {
				rewriteResponseQueryVersionFlag(document.(*machineHelpAPIResponseDocument).ResponseQuery, target.VersionFlag)
				err = applyResponseHelpOptions(document.(*machineHelpAPIResponseDocument), opts)
			}
		} else {
			document, err = service.buildAPI(target.Product, target.Action, target.Version)
			if err == nil {
				action := document.(*machineHelpAPIDocument)
				rewriteResponseQueryVersionFlag(action.ResponseQuery, target.VersionFlag)
				action.GlobalParameters = projectGlobalParameters(ctx.Flags())
				if target.SectionExplicit {
					applyRequestHelpOptions(action, opts, aiMode)
				} else {
					applyActionHelpOptions(action, opts, aiMode, jsonOutput)
					setActionHelpNext(action, target, aiMode)
				}
			}
		}
	case HelpLevelParameter:
		var action *machineHelpAPIDocument
		action, err = service.buildAPI(target.Product, target.Action, target.Version)
		if err == nil {
			action.GlobalParameters = projectGlobalParameters(ctx.Flags())
			var parameter *machineHelpParameterDocument
			parameter, err = buildParameterHelpDocument(action, target.Parameter)
			if err != nil {
				err = invalidMachineHelpParameter(target, action, ctx.Flags(), err)
			}
			if err == nil {
				if target.Operation == HelpOperationSearch {
					searchParameterHelpDocument(parameter, target.SearchQuery)
				}
				document = parameter
			}
		}
	default:
		err = fmt.Errorf("unsupported host Help target %q", target.Level)
	}
	if err != nil {
		var structured *machineHelpError
		if errors.As(err, &structured) {
			if jsonOutput && !aiMode {
				return structured
			}
			return adaptMachineHelpTargetError(target, structured, c.library)
		}
		return err
	}

	c.annotatePluginProvenance(document, target.Product)
	if !aiMode && jsonOutput {
		attachMachineHelpAIModeHint(document)
	}
	if jsonOutput {
		return encodeMachineHelpJSON(ctx.Stdout(), document)
	}
	if err := renderHostHelpText(ctx, document, target.SearchQuery); err != nil {
		return err
	}
	return c.finishCanonicalTextHelp(ctx, aiMode)
}

func rewriteResponseQueryVersionFlag(example *machineHelpQueryExample, versionFlag APIVersionFlag) {
	if example == nil || versionFlag != VersionFlagLegacy {
		return
	}
	example.SchemaCommand = strings.Replace(example.SchemaCommand, " --api-version ", " --version ", 1)
	example.QueryCommand = strings.Replace(example.QueryCommand, " --api-version ", " --version ", 1)
}

func adaptMachineHelpTargetError(target HelpTarget, structured *machineHelpError, library *Library) error {
	if structured == nil {
		return nil
	}
	switch structured.document.Error.Code {
	case "UNKNOWN_PRODUCT":
		return &InvalidProductError{Code: target.Product, library: library}
	case "UNKNOWN_API":
		return &InvalidBaselineCommandError{
			Product: target.Product,
			Command: target.Action,
			Err:     fmt.Errorf("%q is not a valid api. See `aliyun %s --help`.", target.Action, strings.ToLower(target.Product)),
		}
	default:
		return structured
	}
}

func invalidMachineHelpParameter(target HelpTarget, action *machineHelpAPIDocument, flags *cli.FlagSet, cause error) error {
	if action == nil || !strings.Contains(cause.Error(), "unknown parameter flag") {
		return cause
	}
	names := make([]string, 0)
	for _, parameter := range append(activeMachineHelpParameters(action), action.GlobalParameters...) {
		if len(parameter.Options) > 0 {
			for _, option := range parameter.Options {
				names = append(names, strings.TrimPrefix(option, "--"))
			}
			continue
		}
		names = append(names, firstNonEmptyMachineHelpString(parameter.Name, parameter.RawName))
	}
	return &InvalidParameterError{
		Name:           strings.TrimPrefix(target.Parameter, "--"),
		ProductCode:    target.Product,
		ApiName:        target.Action,
		ParameterNames: names,
		flags:          flags,
	}
}

func (c *Commando) buildRootHelpDocument(root *cli.Command) (*machineHelpRootDocument, error) {
	if c.library == nil || c.library.helpRepo == nil {
		return nil, fmt.Errorf("canonical metadata repository is unavailable")
	}
	catalog, err := c.library.helpRepo.GetProducts()
	if err != nil {
		return nil, err
	}
	if len(c.rootCommandHelpSpecs) == 0 && len(c.rootFlagHelpSpecs) == 0 {
		return newMachineHelpService(c.library.helpRepo).buildRoot(root)
	}
	input, err := BuildRootHelpInput(root, catalog, c.rootCommandHelpSpecs, c.rootFlagHelpSpecs)
	if err != nil {
		return nil, err
	}
	document := &machineHelpRootDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "root",
		Target:        machineHelpTarget{Path: []string{"aliyun"}, RequestedStyle: "root"},
		Name:          input.Name,
		Version:       input.Version,
		Description:   machineHelpLocalizedText{EN: input.Description.EN, ZH: input.Description.ZH},
		QuickStart: []string{
			"aliyun configure",
			"aliyun ecs DescribeRegions",
		},
	}
	for _, command := range input.Commands {
		path := append([]string{"aliyun"}, command.Path...)
		document.Commands = append(document.Commands, machineHelpCommandSummary{
			Group:       string(command.Group),
			Path:        path,
			Name:        command.Name,
			Aliases:     append([]string(nil), command.Aliases...),
			Description: machineHelpLocalizedText{EN: command.Description.EN, ZH: command.Description.ZH},
		})
	}
	for _, flag := range input.GlobalFlags {
		shorthand := ""
		if flag.Shorthand != 0 {
			shorthand = "-" + string(flag.Shorthand)
		}
		document.GlobalFlags = append(document.GlobalFlags, machineHelpFlagSummary{
			Name:        "--" + flag.Name,
			Aliases:     append([]string(nil), flag.Aliases...),
			Shorthand:   shorthand,
			Category:    flag.Category,
			Visibility:  string(flag.Visibility),
			Description: machineHelpLocalizedText{EN: flag.Description.EN, ZH: flag.Description.ZH},
		})
	}
	for _, product := range input.Products {
		document.Products = append(document.Products, machineHelpProductSummary{
			Code:          product.Code,
			Name:          machineHelpLocalizedText{EN: product.Name.EN, ZH: product.Name.ZH},
			CommandStyles: []string{"camel", "kebab"},
			CanonicalHelp: true,
			Distribution:  product.Distribution,
		})
	}
	return document, nil
}

func renderHostHelpText(ctx *cli.Context, document any, search string) error {
	switch typed := document.(type) {
	case *machineHelpRootDocument:
		return renderCanonicalRootText(ctx.Stdout(), typed, search)
	case *machineHelpProductDocument:
		return renderCanonicalProductText(ctx.Stdout(), typed, search)
	case *machineHelpAPIDocument:
		if search != "" {
			if err := renderCanonicalRequestSearchText(ctx.Stdout(), typed, search); err != nil {
				return err
			}
			return renderHelpProjectionResult(ctx.Stdout(), "matches", typed.Result, typed.Next)
		}
		return renderCanonicalRequestText(ctx.Stdout(), typed)
	case *machineHelpAPIResponseDocument:
		return renderResponseHelpText(ctx.Stdout(), typed)
	case *machineHelpParameterDocument:
		return renderParameterHelpText(ctx.Stdout(), typed)
	default:
		return fmt.Errorf("unsupported Help document %T", document)
	}
}

func attachMachineHelpAIModeHint(document any) {
	hint := cli.NewAIModeHint()
	value := &machineHelpAIModeHint{Command: hint.Command, Message: hint.Message}
	switch typed := document.(type) {
	case *machineHelpRootDocument:
		typed.AIModeHint = value
	case *machineHelpProductDocument:
		typed.AIModeHint = value
	case *machineHelpAPIDocument:
		typed.AIModeHint = value
	case *machineHelpAPIResponseDocument:
		typed.AIModeHint = value
	case *machineHelpParameterDocument:
		typed.AIModeHint = value
	}
}

func setRootHelpNext(document *machineHelpRootDocument, target HelpTarget, aiMode bool) {
	if document == nil || !document.Result.Truncated {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func setProductHelpNext(document *machineHelpProductDocument, target HelpTarget, aiMode bool) {
	if document == nil || !document.Result.Truncated {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func setActionHelpNext(document *machineHelpAPIDocument, target HelpTarget, aiMode bool) {
	if document == nil || !document.Result.Truncated {
		return
	}
	document.Next = buildHelpNext(target, aiMode)
}

func buildHelpNext(target HelpTarget, aiMode bool) *HelpNext {
	base := target
	base.SectionExplicit = false
	base.Section = HelpSectionRequest
	base.SearchQuery = ""
	if aiMode {
		base.Output = HelpOutputText
	}
	all := base
	all.Operation = HelpOperationAll
	showAll, _ := BuildHelpCommand(all)
	search := base
	search.Operation = HelpOperationSearch
	search.SearchQuery = "<keyword>"
	find, _ := BuildHelpCommand(search)
	return &HelpNext{ShowAll: showAll, Search: find}
}
