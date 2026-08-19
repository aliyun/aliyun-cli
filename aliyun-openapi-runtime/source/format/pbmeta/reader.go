// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pbmeta decodes the indexed protobuf metadata plugin layout.
package pbmeta

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/source/format"
	"github.com/aliyun/aliyun-openapi-runtime/source/format/indexed"
	"github.com/aliyun/aliyun-openapi-runtime/source/storage"
	"google.golang.org/protobuf/proto"
)

const (
	// LayoutName identifies the indexed-protobuf physical layout in MetadataDescriptor.Layout.
	LayoutName   = "indexed-pb"
	DataFileName = "metadata.pb"
)

// Reader uses the shared indexed range reader for bounds, checksum and
// routing, then protobuf-decodes only the selected CommandDefinition payload.
type Reader struct {
	indexed *indexed.Reader
}

// NewReader adds protobuf decoding to an already opened indexed reader.
func NewReader(reader *indexed.Reader) *Reader {
	return &Reader{indexed: reader}
}

func Open(vol storage.Volume, indexFile, dataFile string) (*Reader, error) {
	if dataFile == "" {
		dataFile = DataFileName
	}
	reader, err := indexed.Open(vol, indexFile, dataFile)
	if err != nil {
		return nil, err
	}
	if reader.Index().SchemaVersion != schema.SchemaVersion {
		return nil, fmt.Errorf(
			"unsupported protobuf metadata schemaVersion=%d, want %d",
			reader.Index().SchemaVersion,
			schema.SchemaVersion,
		)
	}
	return NewReader(reader), nil
}

func (r *Reader) VerifyChecksum() error { return r.indexed.VerifyChecksum() }

func (r *Reader) Index() indexed.Index { return r.indexed.Index() }

func (r *Reader) ProductEndpoints() meta.Endpoints {
	return r.indexed.ProductEndpoints()
}

func (r *Reader) APIIndex(product, version string) (*meta.APIIndex, error) {
	return r.indexed.APIIndex(product, version)
}

func (r *Reader) ReadAPI(version, name string) (*meta.API, error) {
	if r.indexed.Index().SchemaVersion != schema.SchemaVersion {
		return nil, fmt.Errorf(
			"unsupported protobuf metadata schemaVersion=%d, want %d",
			r.indexed.Index().SchemaVersion,
			schema.SchemaVersion,
		)
	}
	payload, err := r.indexed.ReadAPI(version, name)
	if err != nil {
		return nil, err
	}
	var definition CommandDefinition
	if err := proto.Unmarshal(payload, &definition); err != nil {
		return nil, fmt.Errorf("decode protobuf api %s/%s: %w", version, name, err)
	}
	canonical, err := toCanonical(&definition)
	if err != nil {
		return nil, fmt.Errorf("decode protobuf api %s/%s: %w", version, name, err)
	}
	return format.DecodeCommandDefinition(canonical, version+"/"+name)
}

func toCanonical(def *CommandDefinition) (*schema.CommandDefinition, error) {
	canonical := &schema.CommandDefinition{
		Name:          def.Name,
		CmdName:       def.CmdName,
		CmdFullName:   def.CmdFullName,
		DescriptionZH: def.DescriptionZh,
		DescriptionEN: def.DescriptionEn,
		Method:        def.Method,
		MultiVersion:  def.MultiVersion,
		Deprecated:    def.Deprecated,
		KebabExample:  def.KebabExample,
		CamelExample:  def.CamelExample,
	}
	if def.Operation != nil {
		canonical.Operation = &schema.OperationConfig{
			Action:          def.Operation.Action,
			APIStyle:        def.Operation.ApiStyle,
			APIVersion:      def.Operation.ApiVersion,
			Method:          def.Operation.Method,
			Protocol:        def.Operation.Protocol,
			URL:             def.Operation.Url,
			IsSSE:           def.Operation.IsSse,
			ReqBodyType:     def.Operation.ReqBodyType,
			ContentType:     def.Operation.ContentType,
			HasWildcardPath: def.Operation.HasWildcardPath,
		}
	}
	parameters, err := toCanonicalArguments(def.Parameters)
	if err != nil {
		return nil, err
	}
	canonical.Parameters = parameters
	return canonical, nil
}

func toCanonicalArguments(values []*Argument) ([]schema.ArgumentDefinition, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]schema.ArgumentDefinition, 0, len(values))
	for index, value := range values {
		if value == nil {
			return nil, fmt.Errorf("nil argument")
		}
		if err := validateArgumentShape(value, fmt.Sprintf("parameters[%d]", index)); err != nil {
			return nil, err
		}
		argument := schema.ArgumentDefinition{
			Name:       value.Name,
			RawName:    value.RawName,
			Type:       value.Type,
			Format:     value.Format,
			Enum:       append([]string(nil), value.Enum...),
			Minimum:    value.Minimum,
			Maximum:    value.Maximum,
			Pattern:    value.Pattern,
			Options:    append([]string(nil), value.Options...),
			HelpZH:     value.HelpZh,
			HelpEN:     value.HelpEn,
			Example:    value.Example,
			Required:   value.Required,
			Location:   value.Location,
			ParamStyle: value.ParamStyle,
			IsWildcard: value.IsWildcard,
		}
		var err error
		if argument.Fields, err = toCanonicalArguments(value.Fields); err != nil {
			return nil, err
		}
		if argument.Element, err = toCanonicalTypeShape(value.Element, fmt.Sprintf("parameters[%d].element", index)); err != nil {
			return nil, err
		}
		if argument.Value, err = toCanonicalTypeShape(value.Value, fmt.Sprintf("parameters[%d].value", index)); err != nil {
			return nil, err
		}
		result = append(result, argument)
	}
	return result, nil
}

func validateArgumentShape(value *Argument, path string) error {
	switch strings.ToLower(strings.TrimSpace(value.Type)) {
	case "array":
		if value.Element == nil {
			return fmt.Errorf("%s: array is missing element", path)
		}
		return validateTypeShape(value.Element, path+".element")
	case "map":
		if value.Value == nil {
			return fmt.Errorf("%s: map is missing value", path)
		}
		return validateTypeShape(value.Value, path+".value")
	case "object":
		for index, field := range value.Fields {
			if field == nil {
				return fmt.Errorf("%s.fields[%d]: nil argument", path, index)
			}
			if err := validateArgumentShape(field, fmt.Sprintf("%s.fields[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTypeShape(value *TypeShape, path string) error {
	switch strings.ToLower(strings.TrimSpace(value.Type)) {
	case "array":
		if value.Element == nil {
			return fmt.Errorf("%s: array is missing element", path)
		}
		return validateTypeShape(value.Element, path+".element")
	case "map":
		if value.Value == nil {
			return fmt.Errorf("%s: map is missing value", path)
		}
		return validateTypeShape(value.Value, path+".value")
	case "object":
		for index, field := range value.Fields {
			if field == nil {
				return fmt.Errorf("%s.fields[%d]: nil argument", path, index)
			}
			if err := validateArgumentShape(field, fmt.Sprintf("%s.fields[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func toCanonicalTypeShape(value *TypeShape, path string) (*schema.TypeShape, error) {
	if value == nil {
		return nil, nil
	}
	shape := &schema.TypeShape{
		Type:    value.Type,
		Format:  value.Format,
		Enum:    append([]string(nil), value.Enum...),
		Minimum: value.Minimum,
		Maximum: value.Maximum,
		Pattern: value.Pattern,
	}
	var err error
	if shape.Fields, err = toCanonicalArguments(value.Fields); err != nil {
		return nil, err
	}
	if shape.Element, err = toCanonicalTypeShape(value.Element, path+".element"); err != nil {
		return nil, err
	}
	if shape.Value, err = toCanonicalTypeShape(value.Value, path+".value"); err != nil {
		return nil, err
	}
	return shape, nil
}
