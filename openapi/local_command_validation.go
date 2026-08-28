package openapi

import (
	"errors"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// validateCanonicalAPICommand performs only metadata-local command identity
// validation. Request validation and execution continue through the existing
// paths after profile resolution.
func (c *Commando) validateCanonicalAPICommand(args []string, ctx *cli.Context) error {
	if len(args) != 2 || c.library == nil || c.library.helpRepo == nil {
		return nil
	}
	method := args[1]
	if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" {
		return nil
	}
	if EstimateCostFlag(ctx.Flags()).IsAssigned() || ForceFlag(ctx.Flags()).IsAssigned() {
		return nil
	}
	version := ""
	if flag, value := rawExplicitVersion(ctx.InvocationArgs()); flag != "" {
		version = value
	}
	resolved, err := newMachineHelpService(c.library.helpRepo).resolveAPI(args[0], args[1], version)
	if err == nil {
		if ctx.UnknownFlags() == nil || resolved == nil || resolved.API == nil {
			return nil
		}
		for _, flag := range ctx.UnknownFlags().Flags() {
			if flag == nil || !flag.IsAssigned() {
				continue
			}
			name := strings.TrimSuffix(flag.Name, "-FILE")
			if resolved.API.FindLegacyParameter(name) == nil {
				return NewInvalidParameterErrorFromCanonical(name, resolved.API, args[0], ctx.Flags())
			}
		}
		return nil
	}
	var structured *machineHelpError
	if !errors.As(err, &structured) {
		// Repository I/O failures should remain on the established execution
		// path instead of making ordinary API execution depend on Help storage.
		return nil
	}
	switch structured.document.Error.Code {
	case "UNKNOWN_API":
		product, _ := c.library.GetProduct(args[0])
		return &InvalidApiError{Name: args[1], product: &product}
	case "UNKNOWN_PRODUCT":
		return &InvalidProductError{Code: args[0], library: c.library}
	default:
		return nil
	}
}

// validateCanonicalRuntimeCommand validates a command for products already
// present in Canonical metadata without preventing unknown/plugin-only
// products from continuing through the established runtime and auto-install
// routing paths.
func (c *Commando) validateCanonicalRuntimeCommand(args []string, ctx *cli.Context) error {
	err := c.validateCanonicalAPICommand(args, ctx)
	var invalidProduct *InvalidProductError
	var invalidParameter *InvalidParameterError
	if errors.As(err, &invalidProduct) || errors.As(err, &invalidParameter) {
		return nil
	}
	return err
}
