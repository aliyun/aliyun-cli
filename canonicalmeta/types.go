package canonicalmeta

// API represents a Canonical API JSON file.
// Fields not needed by the old CLI are omitted from deserialization.
type API struct {
	Name         string `json:"name"`
	CmdName      string `json:"cmd_name,omitempty"`
	CmdFullName  string `json:"cmd_full_name,omitempty"`
	ProductCode  string `json:"product_code,omitempty"`
	MultiVersion bool   `json:"multi_version,omitempty"`
	Deprecated   bool   `json:"deprecated,omitempty"`
	Protocol     string `json:"protocol"`
	Method       string `json:"method"`
	PathPattern  string `json:"pathPattern"`

	Security []string `json:"security,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`

	CamelExample string     `json:"camel_example,omitempty"`
	KebabExample string     `json:"kebab_example,omitempty"`
	Operation    *Operation `json:"operation,omitempty"`

	Parameters []Parameter `json:"parameters"`

	V1Parameters     *[]V1Parameter `json:"v1_parameters,omitempty"`
	V1BodyParameters *[]V1Parameter `json:"v1_body_parameters,omitempty"`
}

// Operation describes the selected HTTP operation for the Canonical command.
type Operation struct {
	Action          string            `json:"action,omitempty"`
	APIStyle        string            `json:"api_style,omitempty"`
	APIVersion      string            `json:"api_version,omitempty"`
	Method          string            `json:"method,omitempty"`
	Protocol        string            `json:"protocol,omitempty"`
	URL             string            `json:"url,omitempty"`
	IsSSE           bool              `json:"is_sse,omitempty"`
	ReqBodyType     string            `json:"req_body_type,omitempty"`
	ContentType     string            `json:"content_type,omitempty"`
	HasWildcardPath bool              `json:"has_wildcard_path,omitempty"`
	BodyMapping     map[string]string `json:"body_mapping,omitempty"`
}

// ProductsIndex is the product catalog stored in metadatas/products.json.
type ProductsIndex struct {
	Products []ProductEntry `json:"products"`
}

// ProductEntry contains product identity, versions, endpoints, and distribution.
type ProductEntry struct {
	Code                 string            `json:"code"`
	Name                 map[string]string `json:"name"`
	APIStyle             string            `json:"api_style"`
	GlobalEndpoint       string            `json:"global_endpoint"`
	RegionalEndpoints    map[string]string `json:"regional_endpoints"`
	RegionalVPCEndpoints map[string]string `json:"regional_vpc_endpoints"`
	LocationServiceCode  string            `json:"location_service_code"`
	PluginDefaultVersion string            `json:"plugin_default_version"`
	Version              string            `json:"version"`
	Versions             []string          `json:"versions"`
	APIs                 []string          `json:"apis"`
	Distribution         string            `json:"distribution,omitempty"`
}

// VersionIndex is the lightweight API index stored beside per-API JSON files.
type VersionIndex struct {
	APIs    map[string]VersionAPIEntry `json:"apis"`
	Style   string                     `json:"style"`
	Version string                     `json:"version"`
}

// VersionAPIEntry is one API entry in version.json.
type VersionAPIEntry struct {
	CmdName       string `json:"cmd_name"`
	Deprecated    bool   `json:"deprecated"`
	DescriptionZh string `json:"description_zh"`
	DescriptionEn string `json:"description_en"`
}

// Description returns the API-level description for the given language.
func (a *API) Description(lang string) string {
	if lang == "zh" {
		return a.DescriptionZh
	}
	return a.DescriptionEn
}

// IsAnonymous returns true if the API's security list contains "Anonymous".
func (a *API) IsAnonymous() bool {
	for _, s := range a.Security {
		if s == "Anonymous" {
			return true
		}
	}
	return false
}

// TypeShape recursively describes an array element or map value.
// Arrays use Element, maps use Value, and objects use Fields.
type TypeShape struct {
	Type    string     `json:"type"`
	Format  string     `json:"format,omitempty"`
	Enum    []string   `json:"enum,omitempty"`
	Minimum string     `json:"minimum,omitempty"`
	Maximum string     `json:"maximum,omitempty"`
	Pattern string     `json:"pattern,omitempty"`
	Fields  []Field    `json:"fields,omitempty"`
	Element *TypeShape `json:"element,omitempty"`
	Value   *TypeShape `json:"value,omitempty"`
}

// Parameter represents a Canonical API parameter.
type Parameter struct {
	Name       string   `json:"name"`
	RawName    string   `json:"raw_name"`
	Type       string   `json:"type"`
	Required   bool     `json:"required"`
	Location   string   `json:"location"`
	ParamStyle string   `json:"param_style,omitempty"`
	IsWildcard bool     `json:"is_wildcard,omitempty"`
	Format     string   `json:"format,omitempty"`
	Enum       []string `json:"enum,omitempty"`
	Minimum    string   `json:"minimum,omitempty"`
	Maximum    string   `json:"maximum,omitempty"`
	Pattern    string   `json:"pattern,omitempty"`
	Example    string   `json:"example,omitempty"`
	Options    []string `json:"options,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
	HelpZh        string `json:"help_zh,omitempty"`
	HelpEn        string `json:"help_en,omitempty"`

	Fields  []Field    `json:"fields,omitempty"`
	Element *TypeShape `json:"element,omitempty"`
	Value   *TypeShape `json:"value,omitempty"`
}

// Help returns the CLI-facing parameter help for the given language.
func (p *Parameter) Help(lang string) string {
	if lang == "zh" {
		return p.HelpZh
	}
	return p.HelpEn
}

// Field represents a nested field within a parameter.
// Fields do not have independent location or param_style.
type Field struct {
	Name     string   `json:"name"`
	RawName  string   `json:"raw_name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Format   string   `json:"format,omitempty"`
	Minimum  string   `json:"minimum,omitempty"`
	Maximum  string   `json:"maximum,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Example  string   `json:"example,omitempty"`
	Enum     []string `json:"enum,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
	HelpZh        string `json:"help_zh,omitempty"`
	HelpEn        string `json:"help_en,omitempty"`

	Fields  []Field    `json:"fields,omitempty"`
	Element *TypeShape `json:"element,omitempty"`
	Value   *TypeShape `json:"value,omitempty"`
}

// Help returns the CLI-facing field help for the given language.
func (f *Field) Help(lang string) string {
	if lang == "zh" {
		return f.HelpZh
	}
	return f.HelpEn
}

// V1Parameter represents a v1_parameters/v1_body_parameters compatibility projection.
type V1Parameter struct {
	Name          string        `json:"name"`
	Position      string        `json:"position"`
	Type          string        `json:"type"`
	ParamStyle    string        `json:"param_style,omitempty"`
	Required      bool          `json:"required"`
	DescriptionZh string        `json:"description_zh,omitempty"`
	DescriptionEn string        `json:"description_en,omitempty"`
	HelpZh        string        `json:"help_zh,omitempty"`
	HelpEn        string        `json:"help_en,omitempty"`
	Example       string        `json:"example,omitempty"`
	SubParameters []V1Parameter `json:"sub_parameters,omitempty"`
}

// HelpOrDescription returns the CLI-facing V1 parameter help for the locale.
func (p *V1Parameter) HelpOrDescription(lang string) string {
	if lang == "zh" {
		if p.HelpZh != "" {
			return p.HelpZh
		}
		return p.DescriptionZh
	}
	if p.HelpEn != "" {
		return p.HelpEn
	}
	return p.DescriptionEn
}
