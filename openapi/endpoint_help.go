package openapi

import (
	"sort"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	runtimemeta "github.com/aliyun/aliyun-openapi-runtime/meta"
)

type machineHelpEndpoint struct {
	RegionID string `json:"regionId,omitempty"`
	Endpoint string `json:"endpoint"`
}

func productHelpEndpoints(product canonicalmeta.ProductEntry, useVPC bool) []machineHelpEndpoint {
	endpoints := runtimemeta.Endpoints{Global: product.GlobalEndpoint, Public: product.RegionalEndpoints, VPC: product.RegionalVPCEndpoints}
	regions := make(map[string]bool)
	for region := range endpoints.Public {
		regions[region] = true
	}
	if useVPC {
		for region := range endpoints.VPC {
			regions[region] = true
		}
	}
	result := make([]machineHelpEndpoint, 0, len(regions)+1)
	for region := range regions {
		if host := endpoints.Resolve(region, useVPC); host != "" {
			result = append(result, machineHelpEndpoint{RegionID: region, Endpoint: host})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RegionID < result[j].RegionID })
	if endpoints.Global != "" {
		result = append(result, machineHelpEndpoint{Endpoint: endpoints.Global})
	}
	return result
}

func (document *machineHelpProductDocument) setCurrentRegion(region string) {
	if region == "" || len(document.Product.Endpoints) == 0 {
		return
	}
	for _, endpoint := range document.Product.Endpoints {
		if endpoint.RegionID == "" || endpoint.RegionID == region {
			return
		}
	}
	document.unsupportedRegion = region
}

// Early parameter/utility Help runs before flags are assigned, so also read
// the original invocation. This does not resolve credentials or make requests.
func helpFlagValue(ctx *cli.Context, name, fallback string) string {
	if flag := ctx.Flags().Get(name); flag != nil {
		if value, assigned := flag.GetValue(); assigned {
			return value
		}
	}
	if value, assigned := recoveryOptionValue(ctx.InvocationArgs(), "--"+name); assigned {
		return value
	}
	return fallback
}
