// Package pbmeta decodes the indexed protobuf metadata plugin layout.
package pbmeta

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun-openapi-runtime/format"
	"github.com/aliyun/aliyun-openapi-runtime/jsonl"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/schema"
	"github.com/aliyun/aliyun-openapi-runtime/storage"
	"google.golang.org/protobuf/proto"
)

const (
	LayoutName    = "indexed-pb"
	LayoutVersion = 1
	DataFileName  = "metadata.pb"
)

// Reader uses the shared indexed range reader for bounds, checksum and
// routing, then protobuf-decodes only the selected CommandDefinition payload.
type Reader struct {
	indexed *jsonl.Reader
}

func Open(vol storage.Volume, indexFile, dataFile string) (*Reader, error) {
	if dataFile == "" {
		dataFile = DataFileName
	}
	indexed, err := jsonl.Open(vol, indexFile, dataFile)
	if err != nil {
		return nil, err
	}
	return &Reader{indexed: indexed}, nil
}

func (r *Reader) VerifyChecksum() error { return r.indexed.VerifyChecksum() }

func (r *Reader) Index() jsonl.Index { return r.indexed.Index() }

func (r *Reader) ProductEndpoints() meta.Endpoints {
	return r.indexed.ProductEndpoints()
}

func (r *Reader) APIIndex(product, version string) (*meta.APIIndex, error) {
	return r.indexed.APIIndex(product, version)
}

func (r *Reader) ReadAPI(version, name string) (*meta.API, error) {
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
			Action:     def.Operation.Action,
			APIStyle:   def.Operation.ApiStyle,
			APIVersion: def.Operation.ApiVersion,
			Method:     def.Operation.Method,
			Protocol:   def.Operation.Protocol,
			URL:        def.Operation.Url,
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
	for _, value := range values {
		if value == nil {
			return nil, fmt.Errorf("nil argument")
		}
		argument := schema.ArgumentDefinition{
			Name:        value.Name,
			RawName:     value.RawName,
			Type:        value.Type,
			Options:     append([]string(nil), value.Options...),
			HelpZH:      value.HelpZh,
			HelpEN:      value.HelpEn,
			Required:    value.Required,
			Location:    value.Location,
			ParamStyle:  value.ParamStyle,
			ElementType: value.ElementType,
			ValueType:   value.ValueType,
		}
		var err error
		if value.HasDefault {
			argument.Default, err = decodeDefault(value.DefaultJson)
			if err != nil {
				return nil, fmt.Errorf("argument %s default: %w", value.Name, err)
			}
		}
		if argument.Fields, err = toCanonicalArguments(value.Fields); err != nil {
			return nil, err
		}
		if argument.ElementFields, err = toCanonicalArguments(value.ElementFields); err != nil {
			return nil, err
		}
		if argument.ValueFields, err = toCanonicalArguments(value.ValueFields); err != nil {
			return nil, err
		}
		result = append(result, argument)
	}
	return result, nil
}

func decodeDefault(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, fmt.Errorf("trailing bytes")
	}
	return value, nil
}
