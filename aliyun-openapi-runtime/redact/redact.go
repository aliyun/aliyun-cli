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

// Package redact exposes the runtime's request-redaction policy to hosts that
// render compatibility output. Keeping one policy prevents legacy and runtime
// dry-run paths from disagreeing about which values are safe to print.
package redact

import internal "github.com/aliyun/aliyun-openapi-runtime/internal"

func IsSensitive(field string) bool { return internal.IsSensitive(field) }

func MaskValue(value string) string { return internal.MaskValue(value) }

func MaskKV(key, value string) string { return internal.MaskKV(key, value) }

func MaskBody(body string) string { return internal.MaskBody(body) }

func MaskAny(data any) any { return internal.MaskAny(data) }
