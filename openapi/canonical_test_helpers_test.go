package openapi

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

type fakeCanonicalRepo struct {
	byName         map[string]*canonicalmeta.API
	versionIndexes map[string]*canonicalmeta.VersionIndex
}

func newFakeCanonicalRepo() *fakeCanonicalRepo {
	return &fakeCanonicalRepo{
		byName:         make(map[string]*canonicalmeta.API),
		versionIndexes: make(map[string]*canonicalmeta.VersionIndex),
	}
}

func (r *fakeCanonicalRepo) AddAPI(productCode, version string, api *canonicalmeta.API) {
	r.byName[apiKey(productCode, version, api.Name)] = api
}

func (r *fakeCanonicalRepo) AddVersionIndex(productCode, version string, index *canonicalmeta.VersionIndex) {
	r.versionIndexes[apiKey(productCode, version, "")] = index
}

func (r *fakeCanonicalRepo) GetAPI(productCode, version, apiName string) (*canonicalmeta.API, error) {
	if api, ok := r.byName[apiKey(productCode, version, apiName)]; ok {
		return api, nil
	}
	return nil, fmt.Errorf("api not found")
}

func (r *fakeCanonicalRepo) GetAPIByPath(productCode, version, method, path string, apiNames []string) (*canonicalmeta.API, error) {
	for _, apiName := range apiNames {
		api, ok := r.byName[apiKey(productCode, version, apiName)]
		if ok && api.MatchLegacyPath(method, path) {
			return api, nil
		}
	}
	return nil, fmt.Errorf("api not found")
}

func (r *fakeCanonicalRepo) GetVersionIndex(productCode, version string) (*canonicalmeta.VersionIndex, error) {
	if index, ok := r.versionIndexes[apiKey(productCode, version, "")]; ok {
		return index, nil
	}
	return nil, fmt.Errorf("version index not found")
}

func apiKey(productCode, version, apiName string) string {
	return strings.ToLower(productCode) + "/" + version + "/" + apiName
}

type testLegacyAPI struct {
	Name        string
	Protocol    string
	Method      string
	PathPattern string
	Product     *meta.Product
	Parameters  []testLegacyParameter
}

type testLegacyParameter struct {
	Name          string
	Position      string
	Type          string
	Description   map[string]string
	Required      bool
	Hidden        bool
	Example       string
	SubParameters []testLegacyParameter
}

func canonicalTestAPI(api *testLegacyAPI) *canonicalmeta.API {
	if api == nil {
		return nil
	}
	result := &canonicalmeta.API{
		Name:        api.Name,
		Protocol:    api.Protocol,
		Method:      api.Method,
		PathPattern: api.PathPattern,
	}
	result.Parameters = canonicalTestParameters(api.Parameters)
	return result
}

func canonicalTestParameters(params []testLegacyParameter) []canonicalmeta.Parameter {
	result := make([]canonicalmeta.Parameter, 0, len(params))
	for _, p := range params {
		cp := canonicalmeta.Parameter{
			Name:          p.Name,
			RawName:       p.Name,
			Type:          canonicalTestType(p.Type),
			Required:      p.Required,
			Location:      strings.ToLower(p.Position),
			DescriptionZh: p.Description["zh"],
			DescriptionEn: p.Description["en"],
			Example:       p.Example,
		}
		if p.Type == "RepeatList" {
			cp.ParamStyle = "repeatList"
		}
		if len(p.SubParameters) > 0 {
			cp.Element = &canonicalmeta.TypeShape{Type: "object"}
		}
		for _, sp := range p.SubParameters {
			cp.Element.Fields = append(cp.Element.Fields, canonicalmeta.Field{
				Name:          sp.Name,
				RawName:       sp.Name,
				Type:          canonicalTestType(sp.Type),
				Required:      sp.Required,
				DescriptionZh: sp.Description["zh"],
				DescriptionEn: sp.Description["en"],
				Example:       sp.Example,
			})
		}
		result = append(result, cp)
	}
	return result
}

func canonicalTestType(t string) string {
	switch t {
	case "RepeatList", "List":
		return "array"
	case "String":
		return "string"
	case "Integer":
		return "int"
	case "Boolean":
		return "bool"
	case "Struct":
		return "map"
	default:
		return strings.ToLower(t)
	}
}
