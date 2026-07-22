package canonicalmeta

// API represents a Canonical API JSON file.
// Fields not needed by the old CLI are omitted from deserialization.
type API struct {
	Name        string `json:"name"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Protocol    string `json:"protocol"`
	Method      string `json:"method"`
	PathPattern string `json:"pathPattern"`

	Security []string `json:"security,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`

	CamelExample string `json:"camel_example,omitempty"`
	KebabExample string `json:"kebab_example,omitempty"`

	Parameters []Parameter `json:"parameters"`

	V1Parameters     *[]V1Parameter `json:"v1_parameters,omitempty"`
	V1BodyParameters *[]V1Parameter `json:"v1_body_parameters,omitempty"`
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

// Parameter represents a Canonical API parameter.
type Parameter struct {
	Name       string `json:"name"`
	RawName    string `json:"raw_name"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	Location   string `json:"location"`
	ParamStyle string `json:"param_style,omitempty"`
	Format     string `json:"format,omitempty"`
	Example    string `json:"example,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
	HelpZh        string `json:"help_zh,omitempty"`
	HelpEn        string `json:"help_en,omitempty"`

	ElementType string `json:"element_type,omitempty"`
	ValueType   string `json:"value_type,omitempty"`

	Fields        []Field `json:"fields,omitempty"`
	ElementFields []Field `json:"element_fields,omitempty"`
	ValueFields   []Field `json:"value_fields,omitempty"`

	InnerElementType   string  `json:"inner_element_type,omitempty"`
	InnerValueType     string  `json:"inner_value_type,omitempty"`
	InnerElementFields []Field `json:"inner_element_fields,omitempty"`
	InnerValueFields   []Field `json:"inner_value_fields,omitempty"`
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
	Name     string `json:"name"`
	RawName  string `json:"raw_name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Example  string `json:"example,omitempty"`

	DescriptionZh string `json:"description_zh,omitempty"`
	DescriptionEn string `json:"description_en,omitempty"`
	HelpZh        string `json:"help_zh,omitempty"`
	HelpEn        string `json:"help_en,omitempty"`

	ElementType string `json:"element_type,omitempty"`
	ValueType   string `json:"value_type,omitempty"`

	Fields        []Field `json:"fields,omitempty"`
	ElementFields []Field `json:"element_fields,omitempty"`
	ValueFields   []Field `json:"value_fields,omitempty"`

	InnerElementType   string  `json:"inner_element_type,omitempty"`
	InnerValueType     string  `json:"inner_value_type,omitempty"`
	InnerElementFields []Field `json:"inner_element_fields,omitempty"`
	InnerValueFields   []Field `json:"inner_value_fields,omitempty"`
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
