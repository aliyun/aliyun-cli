package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/v3/aliyun-openapi-meta"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
)

// LegacyExportMetadata exports canonical metadata in legacy-compatible format.
// Output: outputDir/metadatas/, outputDir/en-US/, outputDir/zh-CN/
func LegacyExportMetadata(outputDir string) error {
	products, err := loadProductsExt()
	if err != nil {
		return fmt.Errorf("load products: %w", err)
	}

	repo := canonicalmeta.NewRepository(aliyunopenapimeta.Metadatas)

	for i := range products.Products {
		p := &products.Products[i]
		fmt.Printf("Exporting %s (%d APIs)...\n", p.Code, len(p.ApiNames))
		if err := exportProduct(repo, p, outputDir); err != nil {
			return fmt.Errorf("export %s: %w", p.Code, err)
		}
	}

	if err := exportMetadatasProductsJSON(outputDir); err != nil {
		return fmt.Errorf("export metadatas/products.json: %w", err)
	}
	if err := exportLocaleProductsJSON(products, "en", path.Join(outputDir, "en-US")); err != nil {
		return fmt.Errorf("export en-US/products.json: %w", err)
	}
	if err := exportLocaleProductsJSON(products, "zh", path.Join(outputDir, "zh-CN")); err != nil {
		return fmt.Errorf("export zh-CN/products.json: %w", err)
	}

	return nil
}

// --- Products loading ---

type legacyExportData struct {
	Type         string                    `json:"type,omitempty"`
	EndpointType string                    `json:"endpoint_type,omitempty"`
	Regions      map[string]legacyRegion   `json:"regions,omitempty"`
	APITitles    map[string]legacyAPITitle `json:"api_titles,omitempty"`
}

type legacyRegion struct {
	RegionID   string            `json:"region_id"`
	AreaID     string            `json:"area_id"`
	RegionName map[string]string `json:"region_name"`
	AreaName   map[string]string `json:"area_name"`
}

type legacyAPITitle struct {
	En string `json:"en"`
	Zh string `json:"zh"`
}

type productExt struct {
	Code                    string            `json:"code"`
	Version                 string            `json:"version"`
	Catalog1                map[string]string `json:"catalog1"`
	Catalog2                map[string]string `json:"catalog2"`
	Name                    map[string]string `json:"name"`
	LocationServiceCode     string            `json:"location_service_code"`
	RegionalEndpoints       map[string]string `json:"regional_endpoints"`
	RegionalVpcEndpoints    map[string]string `json:"regional_vpc_endpoints"`
	GlobalEndpoint          string            `json:"global_endpoint"`
	RegionalEndpointPattern string            `json:"regional_endpoint_patterns"`
	ApiStyle                string            `json:"api_style"`
	ApiNames                []string          `json:"apis"`
	LegacyExport            *legacyExportData `json:"legacy_export,omitempty"`
}

type productSetExt struct {
	Products []productExt `json:"products"`
}

func loadProductsExt() (*productSetExt, error) {
	content, err := readProductsJSON()
	if err != nil {
		return nil, err
	}
	var ps productSetExt
	if err := json.Unmarshal(content, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

func readProductsJSON() ([]byte, error) {
	paths := []string{
		"metadatas/products.json",
		"canonical/metadatas/products.json",
	}
	var lastErr error
	for _, candidate := range paths {
		content, err := aliyunopenapimeta.Metadatas.ReadFile(candidate)
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

// --- metadatas/products.json (strip legacy_export) ---

type cleanProduct struct {
	Code                    string            `json:"code"`
	Version                 string            `json:"version"`
	Catalog1                map[string]string `json:"catalog1,omitempty"`
	Catalog2                map[string]string `json:"catalog2,omitempty"`
	Name                    map[string]string `json:"name,omitempty"`
	LocationServiceCode     string            `json:"location_service_code,omitempty"`
	RegionalEndpoints       map[string]string `json:"regional_endpoints,omitempty"`
	RegionalVpcEndpoints    map[string]string `json:"regional_vpc_endpoints,omitempty"`
	GlobalEndpoint          string            `json:"global_endpoint,omitempty"`
	RegionalEndpointPattern string            `json:"regional_endpoint_patterns,omitempty"`
	ApiStyle                string            `json:"api_style,omitempty"`
	ApiNames                []string          `json:"apis"`
}

type cleanProductSet struct {
	Products []cleanProduct `json:"products"`
}

func exportMetadatasProductsJSON(outputDir string) error {
	ps, err := loadProductsExt()
	if err != nil {
		return err
	}
	clean := cleanProductSet{Products: make([]cleanProduct, len(ps.Products))}
	for i, p := range ps.Products {
		clean.Products[i] = cleanProduct{
			Code:                    p.Code,
			Version:                 p.Version,
			Catalog1:                p.Catalog1,
			Catalog2:                p.Catalog2,
			Name:                    p.Name,
			LocationServiceCode:     p.LocationServiceCode,
			RegionalEndpoints:       p.RegionalEndpoints,
			RegionalVpcEndpoints:    p.RegionalVpcEndpoints,
			GlobalEndpoint:          p.GlobalEndpoint,
			RegionalEndpointPattern: p.RegionalEndpointPattern,
			ApiStyle:                p.ApiStyle,
			ApiNames:                p.ApiNames,
		}
	}
	return writeJSON(path.Join(outputDir, "metadatas", "products.json"), clean)
}

// --- locale products.json ---

type localeProduct struct {
	Code         string                    `json:"code"`
	Name         string                    `json:"name"`
	Version      string                    `json:"version"`
	EndpointType string                    `json:"endpointType"`
	Endpoints    map[string]localeEndpoint `json:"endpoints"`
}

type localeEndpoint struct {
	RegionID   string `json:"regionId"`
	RegionName string `json:"regionName"`
	AreaID     string `json:"areaId"`
	AreaName   string `json:"areaName"`
	Public     string `json:"public,omitempty"`
	Vpc        string `json:"vpc,omitempty"`
}

type localeProductSet struct {
	Products []localeProduct `json:"products"`
}

func exportLocaleProductsJSON(ps *productSetExt, lang string, dir string) error {
	result := localeProductSet{Products: make([]localeProduct, 0, len(ps.Products))}
	for i := range ps.Products {
		p := &ps.Products[i]
		lname := p.Name[lang]
		if lname == "" {
			lname = p.Code
		}
		epType := "regional"
		if p.LegacyExport != nil && p.LegacyExport.EndpointType != "" {
			epType = p.LegacyExport.EndpointType
		}

		endpoints := buildLocaleEndpoints(p, lang)

		result.Products = append(result.Products, localeProduct{
			Code:         p.Code,
			Name:         lname,
			Version:      p.Version,
			EndpointType: epType,
			Endpoints:    endpoints,
		})
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeJSON(path.Join(dir, "products.json"), result)
}

func buildLocaleEndpoints(p *productExt, lang string) map[string]localeEndpoint {
	endpoints := make(map[string]localeEndpoint)

	if p.LegacyExport != nil {
		for regionID, r := range p.LegacyExport.Regions {
			ep := localeEndpoint{
				RegionID:   r.RegionID,
				RegionName: r.RegionName[lang],
				AreaID:     r.AreaID,
				AreaName:   r.AreaName[lang],
			}
			if p.RegionalEndpoints != nil {
				ep.Public = p.RegionalEndpoints[regionID]
			}
			if p.RegionalVpcEndpoints != nil {
				ep.Vpc = p.RegionalVpcEndpoints[regionID]
			}
			endpoints[regionID] = ep
		}
	}

	// Add endpoints from regional_endpoints that aren't in legacy_export.regions
	if p.RegionalEndpoints != nil {
		for regionID, public := range p.RegionalEndpoints {
			if _, exists := endpoints[regionID]; !exists {
				endpoints[regionID] = localeEndpoint{
					RegionID: regionID,
					Public:   public,
				}
			}
		}
	}

	return endpoints
}

// --- version.json ---

type versionFile struct {
	Version string                `json:"version"`
	Style   string                `json:"style"`
	APIs    map[string]versionAPI `json:"apis"`
}

type versionAPI struct {
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Deprecated bool   `json:"deprecated"`
}

func exportVersionJSON(repo *canonicalmeta.Repository, p *productExt, lang string, dir string) error {
	style := p.ApiStyle
	if style == "" {
		style = "rpc"
	}

	vf := versionFile{
		Version: p.Version,
		Style:   style,
		APIs:    make(map[string]versionAPI),
	}

	for _, apiName := range p.ApiNames {
		api, err := repo.GetAPI(p.Code, p.Version, apiName)
		if err != nil {
			return fmt.Errorf("canonical API %s/%s/%s is missing: %w", p.Code, p.Version, apiName, err)
		}

		title := apiName
		if p.LegacyExport != nil && p.LegacyExport.APITitles != nil {
			if t, ok := p.LegacyExport.APITitles[apiName]; ok {
				if lang == "en" && t.En != "" {
					title = t.En
				} else if t.Zh != "" {
					title = t.Zh
				}
			}
		}

		summary := api.Description(lang)

		vf.APIs[apiName] = versionAPI{
			Title:      title,
			Summary:    summary,
			Deprecated: api.Deprecated,
		}
	}

	productDir := path.Join(dir, p.GetLowerCode())
	if err := os.MkdirAll(productDir, 0755); err != nil {
		return err
	}
	return writeJSON(path.Join(productDir, "version.json"), vf)
}

// --- Per-API export ---

func exportProduct(repo *canonicalmeta.Repository, p *productExt, outputDir string) error {
	productCode := p.GetLowerCode()

	for _, apiName := range p.ApiNames {
		api, err := repo.GetAPI(p.Code, p.Version, apiName)
		if err != nil {
			return fmt.Errorf("canonical API %s/%s/%s is missing: %w", p.Code, p.Version, apiName, err)
		}

		// 1. metadatas/<product>/<api>.json
		metaDir := path.Join(outputDir, "metadatas", productCode)
		if err := os.MkdirAll(metaDir, 0755); err != nil {
			return err
		}
		if err := writeJSON(path.Join(metaDir, apiName+".json"), buildMetadatasJSON(api)); err != nil {
			return err
		}

		// 2. en-US/<product>/<api>.json
		enDir := path.Join(outputDir, "en-US", productCode)
		if err := os.MkdirAll(enDir, 0755); err != nil {
			return err
		}
		if err := writeJSON(path.Join(enDir, apiName+".json"), buildLocaleJSON(api, "en")); err != nil {
			return err
		}

		// 3. zh-CN/<product>/<api>.json
		zhDir := path.Join(outputDir, "zh-CN", productCode)
		if err := os.MkdirAll(zhDir, 0755); err != nil {
			return err
		}
		if err := writeJSON(path.Join(zhDir, apiName+".json"), buildLocaleJSON(api, "zh")); err != nil {
			return err
		}
	}

	// Export version.json for both locales
	if err := exportVersionJSON(repo, p, "en", path.Join(outputDir, "en-US")); err != nil {
		return fmt.Errorf("en-US version.json: %w", err)
	}
	if err := exportVersionJSON(repo, p, "zh", path.Join(outputDir, "zh-CN")); err != nil {
		return fmt.Errorf("zh-CN version.json: %w", err)
	}

	return nil
}

// --- Build legacy execution metadata (metadatas/) ---

type legacyExecAPI struct {
	Name        string            `json:"name"`
	Protocol    string            `json:"protocol"`
	Method      string            `json:"method"`
	PathPattern string            `json:"pathPattern"`
	Parameters  []legacyExecParam `json:"parameters"`
}

type legacyExecParam struct {
	Name          string            `json:"name"`
	Position      string            `json:"position"`
	Type          string            `json:"type"`
	Required      bool              `json:"required"`
	SubParameters []legacyExecParam `json:"sub_parameters,omitempty"`
}

func buildMetadatasJSON(api *canonicalmeta.API) *legacyExecAPI {
	result := &legacyExecAPI{
		Name:        api.Name,
		Protocol:    api.Protocol,
		Method:      api.Method,
		PathPattern: api.PathPattern,
	}

	for _, v := range api.LegacyTopLevelParameters() {
		result.Parameters = append(result.Parameters, buildExecParam(v))
	}

	return result
}

func buildExecParam(v *canonicalmeta.LegacyParameterView) legacyExecParam {
	p := legacyExecParam{
		Name:     v.LegacyName(),
		Position: v.LegacyPosition(),
		Type:     legacyDisplayType(v),
		Required: v.LegacyRequired(),
	}

	children := v.LegacyChildren()
	if len(children) > 0 {
		for _, child := range children {
			p.SubParameters = append(p.SubParameters, buildExecParam(child))
		}
	}

	return p
}

// --- Build legacy locale metadata (en-US/zh-CN/) ---

type legacyLocaleAPI struct {
	Name        string              `json:"name"`
	Deprecated  bool                `json:"deprecated"`
	Protocol    string              `json:"protocol"`
	Method      string              `json:"method"`
	PathPattern string              `json:"pathPattern"`
	Parameters  []legacyLocaleParam `json:"parameters"`
}

type legacyLocaleParam struct {
	Name          string              `json:"name"`
	Description   string              `json:"description,omitempty"`
	Position      string              `json:"position"`
	Type          string              `json:"type"`
	Required      bool                `json:"required"`
	SubParameters []legacyLocaleParam `json:"sub_parameters,omitempty"`
}

func buildLocaleJSON(api *canonicalmeta.API, lang string) *legacyLocaleAPI {
	result := &legacyLocaleAPI{
		Name:        api.Name,
		Deprecated:  api.Deprecated,
		Protocol:    api.Protocol,
		Method:      api.Method,
		PathPattern: api.PathPattern,
	}

	for _, v := range api.LegacyTopLevelParameters() {
		result.Parameters = append(result.Parameters, buildLocaleParam(v, lang))
	}

	return result
}

func buildLocaleParam(v *canonicalmeta.LegacyParameterView, lang string) legacyLocaleParam {
	p := legacyLocaleParam{
		Name:        v.LegacyName(),
		Description: v.LegacyDescription(lang),
		Position:    v.LegacyPosition(),
		Type:        legacyDisplayType(v),
		Required:    v.LegacyRequired(),
	}

	children := v.LegacyChildren()
	if len(children) > 0 {
		for _, child := range children {
			p.SubParameters = append(p.SubParameters, buildLocaleParam(child, lang))
		}
	}

	return p
}

// --- Helpers ---

func legacyDisplayType(v *canonicalmeta.LegacyParameterView) string {
	if v.IsLegacyRepeatList() {
		return "RepeatList"
	}
	return v.LegacyType()
}

func writeJSON(filePath string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(b, '\n'), 0666)
}

func (p *productExt) GetLowerCode() string {
	return strings.ToLower(p.Code)
}
