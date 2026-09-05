package openapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

type RootHelpGroup string

const (
	RootGroupCore      RootHelpGroup = "core"
	RootGroupUtils     RootHelpGroup = "utils"
	RootGroupExtension RootHelpGroup = "extension"
	RootGroupProduct   RootHelpGroup = "product"
)

type RootVisibility string

const (
	RootVisibilityDefault  RootVisibility = "default"
	RootVisibilityExtended RootVisibility = "extended"
)

type RootLocalizedText struct {
	EN string `json:"en,omitempty"`
	ZH string `json:"zh,omitempty"`
}

// RootCommandSpec is explicit registration metadata; Path is relative to the
// root command. A utility therefore uses {"utils", "mcp-proxy"}, while its
// compatible hidden root entrypoint is recorded in Aliases only.
type RootCommandSpec struct {
	Path    []string
	Group   RootHelpGroup
	Aliases []string
}

type RootFlagSpec struct {
	Name       string
	Visibility RootVisibility
}

type RootCommandInput struct {
	Group       RootHelpGroup     `json:"group"`
	Path        []string          `json:"path"`
	Name        string            `json:"name"`
	Aliases     []string          `json:"aliases,omitempty"`
	Description RootLocalizedText `json:"description"`
}

type RootFlagInput struct {
	Name        string            `json:"name"`
	Aliases     []string          `json:"aliases,omitempty"`
	Shorthand   rune              `json:"shorthand,omitempty"`
	Category    string            `json:"category,omitempty"`
	Visibility  RootVisibility    `json:"visibility"`
	Description RootLocalizedText `json:"description"`
}

type RootProductInput struct {
	Group        RootHelpGroup     `json:"group"`
	Code         string            `json:"code"`
	Name         RootLocalizedText `json:"name"`
	Distribution string            `json:"distribution,omitempty"`
}

// RootHelpInput is an offline-only document source. Canonical products and
// locally registered command/flag metadata are supplied by the caller; this
// type has no plugin-index or network dependency.
type RootHelpInput struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description RootLocalizedText  `json:"description"`
	Commands    []RootCommandInput `json:"commands"`
	GlobalFlags []RootFlagInput    `json:"globalFlags"`
	Products    []RootProductInput `json:"products"`
}

func BuildRootHelpInput(
	root *cli.Command,
	catalog *canonicalmeta.ProductsIndex,
	commandSpecs []RootCommandSpec,
	flagSpecs []RootFlagSpec,
) (RootHelpInput, error) {
	if root == nil {
		return RootHelpInput{}, fmt.Errorf("root command is nil")
	}
	if catalog == nil {
		return RootHelpInput{}, fmt.Errorf("canonical product catalog is nil")
	}
	if root.Short == nil {
		return RootHelpInput{}, fmt.Errorf("root command description is nil")
	}

	commands, err := buildRootCommandInputs(root, commandSpecs)
	if err != nil {
		return RootHelpInput{}, err
	}
	flags, err := buildRootFlagInputs(root, flagSpecs)
	if err != nil {
		return RootHelpInput{}, err
	}
	products := make([]RootProductInput, 0, len(catalog.Products))
	for _, product := range catalog.Products {
		products = append(products, RootProductInput{
			Group:        RootGroupProduct,
			Code:         strings.ToLower(product.Code),
			Name:         rootLocalizedText(product.Name),
			Distribution: product.Distribution,
		})
	}
	sort.SliceStable(products, func(i, j int) bool { return products[i].Code < products[j].Code })

	return RootHelpInput{
		Name:        root.Name,
		Version:     cli.Version,
		Description: rootLocalizedText(root.Short.GetData()),
		Commands:    commands,
		GlobalFlags: flags,
		Products:    products,
	}, nil
}

func buildRootCommandInputs(root *cli.Command, specs []RootCommandSpec) ([]RootCommandInput, error) {
	topLevelSpecs := make(map[string]bool, len(specs))
	utilsSpecs := make(map[string]bool)
	seenPaths := make(map[string]bool, len(specs))
	commands := make([]RootCommandInput, 0, len(specs))
	for _, spec := range specs {
		if len(spec.Path) == 0 {
			return nil, fmt.Errorf("root command Help spec has an empty path")
		}
		if spec.Group != RootGroupCore && spec.Group != RootGroupUtils && spec.Group != RootGroupExtension {
			return nil, fmt.Errorf("root command %q has invalid Help group %q", strings.Join(spec.Path, " "), spec.Group)
		}
		pathKey := strings.Join(spec.Path, " ")
		if seenPaths[pathKey] {
			return nil, fmt.Errorf("root command Help spec %q duplicated", pathKey)
		}
		seenPaths[pathKey] = true
		command := findRootCommand(root, spec.Path)
		if command == nil {
			return nil, fmt.Errorf("root command Help spec %q is not registered", pathKey)
		}
		if command.Hidden {
			return nil, fmt.Errorf("hidden root command %q cannot be a Help document entry", pathKey)
		}
		if len(spec.Path) == 1 {
			topLevelSpecs[spec.Path[0]] = true
		}
		if len(spec.Path) == 2 && spec.Path[0] == "utils" {
			utilsSpecs[spec.Path[1]] = true
		}
		commands = append(commands, RootCommandInput{
			Group:       spec.Group,
			Path:        append([]string(nil), spec.Path...),
			Name:        pathKey,
			Aliases:     append([]string(nil), spec.Aliases...),
			Description: commandDescription(command),
		})
	}

	for _, name := range root.SubCommandNames() {
		command := root.GetSubCommand(name)
		if command == nil || command.Hidden {
			continue
		}
		if !topLevelSpecs[name] {
			return nil, fmt.Errorf("visible root command %q has no explicit Help group", name)
		}
	}
	if utils := root.GetSubCommand("utils"); utils != nil {
		for _, name := range utils.SubCommandNames() {
			command := utils.GetSubCommand(name)
			if command == nil || command.Hidden {
				continue
			}
			if !utilsSpecs[name] {
				return nil, fmt.Errorf("visible utility command %q has no explicit Help group", name)
			}
		}
	}
	return commands, nil
}

func buildRootFlagInputs(root *cli.Command, specs []RootFlagSpec) ([]RootFlagInput, error) {
	seen := make(map[string]bool, len(specs))
	flags := make([]RootFlagInput, 0, len(specs))
	for _, spec := range specs {
		if seen[spec.Name] {
			return nil, fmt.Errorf("root flag Help spec %q duplicated", spec.Name)
		}
		seen[spec.Name] = true
		if spec.Visibility != RootVisibilityDefault && spec.Visibility != RootVisibilityExtended {
			return nil, fmt.Errorf("root flag %q has invalid Help visibility %q", spec.Name, spec.Visibility)
		}
		flag := root.Flags().Get(spec.Name)
		if spec.Name == "help" {
			flag = cli.NewHelpFlag()
		}
		if flag == nil {
			return nil, fmt.Errorf("root flag Help spec %q is not registered", spec.Name)
		}
		if flag.Hidden {
			return nil, fmt.Errorf("hidden root flag %q cannot be a Help document entry", spec.Name)
		}
		flags = append(flags, RootFlagInput{
			Name:        flag.Name,
			Aliases:     append([]string(nil), flag.Aliases...),
			Shorthand:   flag.Shorthand,
			Category:    flag.Category,
			Visibility:  spec.Visibility,
			Description: flagDescription(flag),
		})
	}
	for _, flag := range root.Flags().Flags() {
		if flag == nil || flag.Hidden {
			continue
		}
		if !seen[flag.Name] {
			return nil, fmt.Errorf("visible root flag %q has no explicit Help visibility", flag.Name)
		}
	}
	return flags, nil
}

func findRootCommand(root *cli.Command, path []string) *cli.Command {
	command := root
	for _, name := range path {
		command = command.GetSubCommand(name)
		if command == nil {
			return nil
		}
	}
	return command
}

func commandDescription(command *cli.Command) RootLocalizedText {
	if command == nil || command.Short == nil {
		return RootLocalizedText{}
	}
	return rootLocalizedText(command.Short.GetData())
}

func flagDescription(flag *cli.Flag) RootLocalizedText {
	if flag == nil || flag.Short == nil {
		return RootLocalizedText{}
	}
	return rootLocalizedText(flag.Short.GetData())
}

func rootLocalizedText(values map[string]string) RootLocalizedText {
	return RootLocalizedText{EN: values["en"], ZH: values["zh"]}
}
