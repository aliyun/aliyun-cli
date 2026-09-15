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

package pbmeta

import "testing"

// These tests exercise the generated protobuf compatibility surface. The nil
// receiver checks are intentional: protobuf getters promise zero values for
// absent nested messages, and the runtime reader relies on that behavior.
func TestGeneratedCommandDefinitionAccessors(t *testing.T) {
	operation := &Operation{Action: "Run"}
	parameter := &Argument{Name: "name"}
	message := &CommandDefinition{
		ProductCode: "ecs", Name: "RunInstances", CmdName: "run-instances", CmdFullName: "ecs run-instances",
		DescriptionZh: "中文", DescriptionEn: "English", Method: "POST", MultiVersion: true, Deprecated: true,
		KebabExample: "--name value", CamelExample: "--Name value", Operation: operation, Parameters: []*Argument{parameter},
	}
	if message.GetProductCode() != "ecs" || message.GetName() != "RunInstances" || message.GetCmdName() != "run-instances" ||
		message.GetCmdFullName() != "ecs run-instances" || message.GetDescriptionZh() != "中文" || message.GetDescriptionEn() != "English" ||
		message.GetMethod() != "POST" || !message.GetMultiVersion() || !message.GetDeprecated() ||
		message.GetKebabExample() != "--name value" || message.GetCamelExample() != "--Name value" ||
		message.GetOperation() != operation || len(message.GetParameters()) != 1 {
		t.Fatalf("generated getters returned unexpected values: %#v", message)
	}
	if message.String() == "" || message.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("generated protobuf reflection returned empty metadata")
	}
	message.ProtoMessage()
	if descriptor, indexes := message.Descriptor(); len(descriptor) == 0 || len(indexes) != 1 || indexes[0] != 0 {
		t.Fatalf("Descriptor() = %d bytes, %v", len(descriptor), indexes)
	}

	var zero *CommandDefinition
	if zero.GetProductCode() != "" || zero.GetName() != "" || zero.GetCmdName() != "" || zero.GetCmdFullName() != "" ||
		zero.GetDescriptionZh() != "" || zero.GetDescriptionEn() != "" || zero.GetMethod() != "" || zero.GetMultiVersion() || zero.GetDeprecated() ||
		zero.GetKebabExample() != "" || zero.GetCamelExample() != "" || zero.GetOperation() != nil || zero.GetParameters() != nil {
		t.Fatal("nil CommandDefinition getters did not return zero values")
	}
	if zero.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("nil CommandDefinition reflection returned empty descriptor")
	}

	message.Reset()
	if message.GetName() != "" || message.GetOperation() != nil {
		t.Fatalf("Reset() left data behind: %#v", message)
	}
}

func TestGeneratedOperationAccessors(t *testing.T) {
	message := &Operation{
		Action: "Run", ApiStyle: "ROA", ApiVersion: "v1", Method: "POST", Protocol: "HTTPS", Url: "/run",
		IsSse: true, ReqBodyType: "json", ContentType: "application/json", HasWildcardPath: true,
	}
	if message.GetAction() != "Run" || message.GetApiStyle() != "ROA" || message.GetApiVersion() != "v1" ||
		message.GetMethod() != "POST" || message.GetProtocol() != "HTTPS" || message.GetUrl() != "/run" ||
		!message.GetIsSse() || message.GetReqBodyType() != "json" || message.GetContentType() != "application/json" || !message.GetHasWildcardPath() {
		t.Fatalf("generated getters returned unexpected values: %#v", message)
	}
	if message.String() == "" || message.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("generated protobuf reflection returned empty metadata")
	}
	message.ProtoMessage()
	if descriptor, indexes := message.Descriptor(); len(descriptor) == 0 || len(indexes) != 1 || indexes[0] != 1 {
		t.Fatalf("Descriptor() = %d bytes, %v", len(descriptor), indexes)
	}

	var zero *Operation
	if zero.GetAction() != "" || zero.GetApiStyle() != "" || zero.GetApiVersion() != "" || zero.GetMethod() != "" ||
		zero.GetProtocol() != "" || zero.GetUrl() != "" || zero.GetIsSse() || zero.GetReqBodyType() != "" ||
		zero.GetContentType() != "" || zero.GetHasWildcardPath() {
		t.Fatal("nil Operation getters did not return zero values")
	}
	if zero.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("nil Operation reflection returned empty descriptor")
	}

	message.Reset()
	if message.GetAction() != "" || message.GetIsSse() {
		t.Fatalf("Reset() left data behind: %#v", message)
	}
}

func TestGeneratedArgumentAccessors(t *testing.T) {
	field := &Argument{Name: "field"}
	element := &TypeShape{Type: "string"}
	value := &TypeShape{Type: "integer"}
	message := &Argument{
		Name: "items", RawName: "Items", Type: "array", Options: []string{"--items"}, HelpZh: "中文", HelpEn: "English",
		Required: true, Location: "query", ParamStyle: "repeatList", Fields: []*Argument{field}, Element: element, Value: value,
		Example: "one", IsWildcard: true, Enum: []string{"one"}, Minimum: "1", Maximum: "10", Pattern: "^[a-z]+$",
		Format: "int32", MinLength: "1", MaxLength: "8", DocRequired: true, DirectBody: true,
	}
	if message.GetName() != "items" || message.GetRawName() != "Items" || message.GetType() != "array" || len(message.GetOptions()) != 1 ||
		message.GetHelpZh() != "中文" || message.GetHelpEn() != "English" || !message.GetRequired() || message.GetLocation() != "query" ||
		message.GetParamStyle() != "repeatList" || len(message.GetFields()) != 1 || message.GetElement() != element || message.GetValue() != value ||
		message.GetExample() != "one" || !message.GetIsWildcard() || len(message.GetEnum()) != 1 || message.GetMinimum() != "1" ||
		message.GetMaximum() != "10" || message.GetPattern() != "^[a-z]+$" || message.GetFormat() != "int32" ||
		message.GetMinLength() != "1" || message.GetMaxLength() != "8" || !message.GetDocRequired() || !message.GetDirectBody() {
		t.Fatalf("generated getters returned unexpected values: %#v", message)
	}
	if message.String() == "" || message.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("generated protobuf reflection returned empty metadata")
	}
	message.ProtoMessage()
	if descriptor, indexes := message.Descriptor(); len(descriptor) == 0 || len(indexes) != 1 || indexes[0] != 2 {
		t.Fatalf("Descriptor() = %d bytes, %v", len(descriptor), indexes)
	}

	var zero *Argument
	if zero.GetName() != "" || zero.GetRawName() != "" || zero.GetType() != "" || zero.GetOptions() != nil || zero.GetHelpZh() != "" ||
		zero.GetHelpEn() != "" || zero.GetRequired() || zero.GetLocation() != "" || zero.GetParamStyle() != "" || zero.GetFields() != nil ||
		zero.GetElement() != nil || zero.GetValue() != nil || zero.GetExample() != "" || zero.GetIsWildcard() || zero.GetEnum() != nil ||
		zero.GetMinimum() != "" || zero.GetMaximum() != "" || zero.GetPattern() != "" || zero.GetFormat() != "" ||
		zero.GetMinLength() != "" || zero.GetMaxLength() != "" || zero.GetDocRequired() || zero.GetDirectBody() {
		t.Fatal("nil Argument getters did not return zero values")
	}
	if zero.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("nil Argument reflection returned empty descriptor")
	}

	message.Reset()
	if message.GetName() != "" || message.GetElement() != nil || message.GetRequired() {
		t.Fatalf("Reset() left data behind: %#v", message)
	}
}

func TestGeneratedTypeShapeAccessors(t *testing.T) {
	field := &Argument{Name: "field"}
	element := &TypeShape{Type: "string"}
	value := &TypeShape{Type: "integer"}
	message := &TypeShape{
		Type: "map", Fields: []*Argument{field}, Element: element, Value: value, Enum: []string{"one"},
		Minimum: "1", Maximum: "10", Pattern: "^[a-z]+$", Format: "int32", MinLength: "1", MaxLength: "8",
	}
	if message.GetType() != "map" || len(message.GetFields()) != 1 || message.GetElement() != element || message.GetValue() != value ||
		len(message.GetEnum()) != 1 || message.GetMinimum() != "1" || message.GetMaximum() != "10" ||
		message.GetPattern() != "^[a-z]+$" || message.GetFormat() != "int32" || message.GetMinLength() != "1" || message.GetMaxLength() != "8" {
		t.Fatalf("generated getters returned unexpected values: %#v", message)
	}
	if message.String() == "" || message.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("generated protobuf reflection returned empty metadata")
	}
	message.ProtoMessage()
	if descriptor, indexes := message.Descriptor(); len(descriptor) == 0 || len(indexes) != 1 || indexes[0] != 3 {
		t.Fatalf("Descriptor() = %d bytes, %v", len(descriptor), indexes)
	}

	var zero *TypeShape
	if zero.GetType() != "" || zero.GetFields() != nil || zero.GetElement() != nil || zero.GetValue() != nil || zero.GetEnum() != nil ||
		zero.GetMinimum() != "" || zero.GetMaximum() != "" || zero.GetPattern() != "" || zero.GetFormat() != "" ||
		zero.GetMinLength() != "" || zero.GetMaxLength() != "" {
		t.Fatal("nil TypeShape getters did not return zero values")
	}
	if zero.ProtoReflect().Descriptor().FullName() == "" {
		t.Fatal("nil TypeShape reflection returned empty descriptor")
	}

	message.Reset()
	if message.GetType() != "" || message.GetElement() != nil {
		t.Fatalf("Reset() left data behind: %#v", message)
	}
}
