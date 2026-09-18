package openapi

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

type machineHelpEndpoint struct {
	RegionID    string `json:"regionId,omitempty"`
	RegionName  string `json:"regionName,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	VPCEndpoint string `json:"vpcEndpoint,omitempty"`
}

func productHelpEndpoints(product canonicalmeta.ProductEntry) []machineHelpEndpoint {
	regions := make(map[string]bool)
	for region := range product.RegionalEndpoints {
		regions[region] = true
	}
	for region := range product.RegionalVPCEndpoints {
		regions[region] = true
	}
	result := make([]machineHelpEndpoint, 0, len(regions)+1)
	for region := range regions {
		public, vpc := product.RegionalEndpoints[region], product.RegionalVPCEndpoints[region]
		if public != "" || vpc != "" {
			result = append(result, machineHelpEndpoint{RegionID: region,
				RegionName: product.RegionNames[region][localizedMachineHelpLanguage()], Endpoint: public, VPCEndpoint: vpc})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RegionID < result[j].RegionID })
	if product.GlobalEndpoint != "" {
		result = append(result, machineHelpEndpoint{Endpoint: product.GlobalEndpoint})
	}
	return result
}

func (document *machineHelpProductDocument) setCurrentRegion(region string) {
	if region == "" || len(document.Endpoints) == 0 {
		return
	}
	for _, endpoint := range document.Endpoints {
		if endpoint.RegionID == "" || (endpoint.RegionID == region &&
			(endpoint.Endpoint != "" || document.useVPCEndpoint && endpoint.VPCEndpoint != "")) {
			return
		}
	}
	document.unsupportedRegion = region
}

func renderProductEndpoints(w io.Writer, document *machineHelpProductDocument) error {
	if _, err := fmt.Fprintln(w, "\nENDPOINTS"); err != nil {
		return err
	}
	if document.unsupportedRegion != "" {
		if _, err := fmt.Fprintf(w, "Note: current region %s is not supported by this product.\n", document.unsupportedRegion); err != nil {
			return err
		}
	}
	rows := [][4]string{{"RegionId", "RegionName", "Endpoint", "VpcEndpoint"}}
	for _, endpoint := range document.Endpoints {
		region := endpoint.RegionID
		if region == "" {
			region = "(global)"
		}
		rows = append(rows, [4]string{region, endpoint.RegionName, endpoint.Endpoint, endpoint.VPCEndpoint})
	}
	var widths [4]int
	for i := range rows {
		for j, cell := range rows[i] {
			if cell == "" {
				rows[i][j] = "-"
			}
			widths[j] = max(widths[j], endpointCellWidth(rows[i][j]))
		}
	}
	for _, row := range rows {
		line := "  "
		for column, cell := range row {
			line += cell
			if column < len(row)-1 {
				line += strings.Repeat(" ", widths[column]-endpointCellWidth(cell)+2)
			}
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// ponytail: region labels are Latin/CJK; use a grapheme-width library if they
// later include emoji sequences. fmt/tabwriter alone misalign CJK labels.
func endpointCellWidth(text string) int {
	width := 0
	for _, r := range text {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		width++
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) ||
			(r >= '\uff01' && r <= '\uff60') || (r >= '\uffe0' && r <= '\uffe6') {
			width++
		}
	}
	return width
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
