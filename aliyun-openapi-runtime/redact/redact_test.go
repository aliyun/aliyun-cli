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

package redact

import "testing"

func TestPublicRedactionContract(t *testing.T) {
	if !IsSensitive("Password") || IsSensitive("RegionId") {
		t.Fatal("unexpected sensitive-field classification")
	}
	if got := MaskValue("secret-value"); got != "secr***" {
		t.Fatalf("MaskValue = %q", got)
	}
	if got := MaskKV("Password", "secret-value"); got != "secr***" {
		t.Fatalf("MaskKV = %q", got)
	}
	if got := MaskBody(`{"password":"secret-value"}`); got != `{"password":"secr***"}` {
		t.Fatalf("MaskBody = %q", got)
	}
	masked := MaskAny(map[string]any{"token": "secret-value"}).(map[string]any)
	if masked["token"] != "secr***" {
		t.Fatalf("MaskAny = %#v", masked)
	}
}
