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

package engine

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type dispatchErrorTestSource struct{}

func (dispatchErrorTestSource) Kind() source.Kind { return source.KindBaseline }

func (dispatchErrorTestSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	if code != "demo" {
		return nil, nil, source.ErrNotFound
	}
	return &meta.Product{
		Code: "demo", Versions: []string{"2024-01-01"}, DefaultVersion: "2024-01-01",
	}, &source.Provenance{Kind: source.KindBaseline}, nil
}

func (dispatchErrorTestSource) LoadAPIIndex(code, version string) (*meta.APIIndex, error) {
	if code != "demo" || version != "2024-01-01" {
		return nil, source.ErrNotFound
	}
	index := &meta.APIIndex{
		ProductCode: code,
		Version:     version,
		Entries: map[string]meta.APIIndexEntry{
			"RunThing": {APIName: "RunThing", CmdName: "run-thing"},
		},
	}
	index.BuildCmdIndex()
	return index, nil
}

func (dispatchErrorTestSource) LoadAPI(code, version, name string) (*meta.API, error) {
	if code != "demo" || version != "2024-01-01" || name != "RunThing" {
		return nil, source.ErrNotFound
	}
	return &meta.API{
		Name: "RunThing", CmdName: "run-thing", ProductCode: "demo",
		Version: "2024-01-01", Method: "POST", Protocol: "HTTPS", Style: meta.StyleRPC,
		Endpoints: meta.Endpoints{Global: "demo.example.com"},
		Parameters: []meta.Parameter{
			{
				Name: "instance_type", RawName: "InstanceType", Type: meta.TypeString,
				Position: meta.PosQuery, Required: true, Options: []string{"--instance-type"},
				Enum: []string{"ecs.g6"},
			},
			{
				Name: "change_reason", RawName: "ChangeReason", Type: meta.TypeString,
				Position: meta.PosQuery, DocRequired: true, Options: []string{"--change-reason"},
			},
		},
	}, nil
}

func newDispatchErrorTestEngine() *Engine {
	return NewEngine(func() (loader.Loader, error) {
		return loader.New(dispatchErrorTestSource{}), nil
	}, nil)
}

func TestDispatchPreservesUnknownFlagDetails(t *testing.T) {
	err := newDispatchErrorTestEngine().Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instnace-type", "ecs.g6"},
		Out:  io.Discard,
	})

	var unknownFlag *argparser.UnknownFlagError
	if !errors.As(err, &unknownFlag) {
		t.Fatalf("Dispatch error %T does not preserve UnknownFlagError: %v", err, err)
	}
	if unknownFlag.Flag != "instnace-type" || !reflect.DeepEqual(unknownFlag.Known, []string{"change-reason", "instance-type"}) {
		t.Fatalf("UnknownFlagError = %#v", unknownFlag)
	}
}

func TestDispatchPreservesMissingRequiredDetails(t *testing.T) {
	err := newDispatchErrorTestEngine().Dispatch(Request{
		Args: []string{"demo", "run-thing"},
		Out:  io.Discard,
	})
	if got, want := err.Error(), "missing required parameter(s): --instance-type"; got != want {
		t.Fatalf("Dispatch error text = %q, want %q", got, want)
	}

	var usage *UsageError
	if !errors.As(err, &usage) || usage.Code != "MISSING_REQUIRED_PARAMETER" {
		t.Fatalf("Dispatch error = %#v, want MISSING_REQUIRED_PARAMETER UsageError", err)
	}
	var missing *runtime.MissingRequiredError
	if !errors.As(err, &missing) || !reflect.DeepEqual(missing.Flags, []string{"--instance-type"}) {
		t.Fatalf("Dispatch error does not preserve MissingRequiredError: %v", err)
	}
}

func TestDispatchConstraintValidationFollowsAIMode(t *testing.T) {
	args := []string{"demo", "run-thing", "--instance-type", "ecs.invalid", "--change-reason", "testing"}

	err := newDispatchErrorTestEngine().Dispatch(Request{
		Args: args, Out: io.Discard, AIMode: true,
	})
	var usage *UsageError
	if !errors.As(err, &usage) || usage.Code != "INVALID_PARAMETER_VALUE" {
		t.Fatalf("AI mode error = %#v, want INVALID_PARAMETER_VALUE UsageError", err)
	}
	var violation *runtime.ConstraintViolationError
	if !errors.As(err, &violation) || violation.Flag != "--instance-type" ||
		violation.Constraint != "enum" {
		t.Fatalf("AI mode did not preserve constraint violation: %v", err)
	}

	credentialCause := errors.New("stop after validation")
	err = newDispatchErrorTestEngine().Dispatch(Request{
		Args: args, Out: io.Discard,
		Host: runtime.StaticHost{CredErr: credentialCause},
	})
	if errors.As(err, &usage) && usage.Code == "INVALID_PARAMETER_VALUE" {
		t.Fatalf("human mode unexpectedly validated metadata constraints: %v", err)
	}
	if errors.As(err, &violation) {
		t.Fatalf("human mode unexpectedly returned constraint violation: %v", err)
	}
	if !errors.Is(err, credentialCause) {
		t.Fatalf("human mode did not continue past constraints: %v", err)
	}
}

func TestDispatchDocRequiredValidationFollowsAIModeAndPrecedesConstraints(t *testing.T) {
	args := []string{"demo", "run-thing", "--instance-type", "ecs.invalid"}

	err := newDispatchErrorTestEngine().Dispatch(Request{
		Args: args, Out: io.Discard, AIMode: true,
	})
	var usage *UsageError
	if !errors.As(err, &usage) || usage.Code != "MISSING_REQUIRED_PARAMETER" {
		t.Fatalf("AI mode error = %#v, want MISSING_REQUIRED_PARAMETER UsageError", err)
	}
	var missing *runtime.MissingRequiredError
	if !errors.As(err, &missing) ||
		!reflect.DeepEqual(missing.Flags, []string{"--change-reason"}) ||
		!reflect.DeepEqual(missing.Paths, []string{"ChangeReason"}) {
		t.Fatalf("AI mode did not preserve docRequired details: %v", err)
	}
	var violation *runtime.ConstraintViolationError
	if errors.As(err, &violation) {
		t.Fatalf("constraints ran before docRequired validation: %v", err)
	}

	credentialCause := errors.New("stop after validation")
	err = newDispatchErrorTestEngine().Dispatch(Request{
		Args: args, Out: io.Discard,
		Host: runtime.StaticHost{CredErr: credentialCause},
	})
	if errors.As(err, &missing) {
		t.Fatalf("human mode unexpectedly validated docRequired metadata: %v", err)
	}
	if !errors.Is(err, credentialCause) {
		t.Fatalf("human mode did not continue past docRequired validation: %v", err)
	}
}

func TestDispatchPreservesCredentialFailure(t *testing.T) {
	cause := errors.New("credential unavailable")
	err := newDispatchErrorTestEngine().Dispatch(Request{
		Args: []string{"demo", "run-thing", "--instance-type", "ecs.g6"},
		Out:  io.Discard,
		Host: runtime.StaticHost{CredErr: cause},
	})

	var credential *CredentialError
	if !errors.As(err, &credential) {
		t.Fatalf("Dispatch error %T does not preserve CredentialError: %v", err, err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("Dispatch error does not preserve credential cause: %v", err)
	}
}

func TestBuildExecContextRawBodyPrecedesBodyFile(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{
		Body:        `{"from":"body"}`,
		BodySet:     true,
		BodyFile:    "/file/that/must/not/be/read",
		BodyFileSet: true,
		DryRun:      true,
	}}
	ec, err := buildExecContext(Request{}, &meta.API{}, res)
	if err != nil {
		t.Fatalf("buildExecContext: %v", err)
	}
	if ec.RawBody != res.Reserved.Body {
		t.Fatalf("raw body = %#v, want %#v", ec.RawBody, res.Reserved.Body)
	}
}

func TestBuildExecContextPreservesExplicitEmptyRawBody(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{BodySet: true, DryRun: true}}
	ec, err := buildExecContext(Request{}, &meta.API{}, res)
	if err != nil {
		t.Fatalf("buildExecContext: %v", err)
	}
	if ec.RawBody == nil || ec.RawBody != "" {
		t.Fatalf("raw body = %#v, want explicit empty string", ec.RawBody)
	}
}

func TestApplyMetadataPluginProvenance(t *testing.T) {
	ec := &runtime.ExecContext{}
	applyMetadataPluginProvenance(ec, &source.Provenance{
		PluginName:    "aliyun-cli-fc",
		PluginVersion: "0.7.1",
	})

	if ec.MetadataPluginName != "aliyun-cli-fc" || ec.MetadataPluginVersion != "0.7.1" {
		t.Fatalf("metadata plugin identity = %q/%q", ec.MetadataPluginName, ec.MetadataPluginVersion)
	}
}

func TestValidateDispatchOptionsRequiresEstimateCost(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{
		EstimateCostContext: []string{"Traffic=10"},
	}}
	err := validateDispatchOptions(res)
	if err == nil || err.Error() != "--estimate-cost-context requires --estimate-cost" {
		t.Fatalf("validateDispatchOptions error = %v", err)
	}

	res.Reserved.EstimateCost = true
	if err := validateDispatchOptions(res); err != nil {
		t.Fatalf("validateDispatchOptions with --estimate-cost: %v", err)
	}
}

func TestValidateDispatchOptionsPreservesTypedConflict(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{
		DryRunJSON: true,
		Pager:      &argparser.PagerConfig{},
	}}
	err := validateDispatchOptions(res)
	var conflict *InvalidOptionCombinationError
	if !errors.As(err, &conflict) {
		t.Fatalf("validateDispatchOptions error %T does not preserve conflict: %v", err, err)
	}
	if !reflect.DeepEqual(conflict.Options, []string{"--cli-dry-run-json", "--pager"}) {
		t.Fatalf("conflict options = %#v", conflict.Options)
	}
	if got, want := err.Error(), "--cli-dry-run-json cannot be used with --pager"; got != want {
		t.Fatalf("conflict text = %q, want %q", got, want)
	}
}

func TestBuildExecContextPreservesTypedHeaderAndBodyFileErrors(t *testing.T) {
	t.Run("header", func(t *testing.T) {
		_, err := buildExecContext(Request{}, &meta.API{}, &argparser.Result{Reserved: argparser.Reserved{
			Headers: []string{"broken"}, DryRun: true,
		}})
		var invalid *InvalidHeaderError
		if !errors.As(err, &invalid) {
			t.Fatalf("buildExecContext error %T does not preserve header: %v", err, err)
		}
		if invalid.Input != "broken" || invalid.ExpectedFormat != "Name=Value" {
			t.Fatalf("InvalidHeaderError = %#v", invalid)
		}
	})

	t.Run("body file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.json")
		_, err := buildExecContext(Request{}, &meta.API{}, &argparser.Result{Reserved: argparser.Reserved{
			BodyFile: path, BodyFileSet: true, DryRun: true,
		}})
		var invalid *InvalidBodyFileError
		if !errors.As(err, &invalid) {
			t.Fatalf("buildExecContext error %T does not preserve body file: %v", err, err)
		}
		if invalid.Path != path || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("InvalidBodyFileError = %#v, err = %v", invalid, err)
		}
	})
}
