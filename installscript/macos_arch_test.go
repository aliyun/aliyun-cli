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

package installscript

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMacOSPackageArch(t *testing.T) {
	cases := []struct {
		machine string
		want    string
	}{
		{"arm64", "arm64"},
		{"x86_64", "amd64"},
		{"i386", "universal"},
		{"", "universal"},
		{"unknown", "universal"},
		{"aarch64", "universal"}, // macOS uses arm64; aarch64 falls back
	}
	for _, tc := range cases {
		t.Run(tc.machine, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveMacOSPackageArch(tc.machine))
		})
	}
}

func TestResolveMacOSPackageArch_MatchesInstallSh(t *testing.T) {
	installSh := filepath.Join("..", "install.sh")
	_, err := os.Stat(installSh)
	require.NoError(t, err)

	machines := []string{"arm64", "x86_64", "i386", "", "powerpc", "aarch64"}
	for _, machine := range machines {
		t.Run(machine, func(t *testing.T) {
			cmd := exec.Command("bash", "-c",
				`ALIYUN_CLI_INSTALL_SH_LIBONLY=1 source "$1"; resolve_macos_package_arch "$2"`,
				"bash", installSh, machine)
			out, err := cmd.Output()
			require.NoError(t, err, "bash resolve_macos_package_arch %q", machine)
			got := strings.TrimSpace(string(out))
			assert.Equal(t, ResolveMacOSPackageArch(machine), got)
		})
	}
}

func TestInstallShDownloadsResolvedArchPackage(t *testing.T) {
	installSh := filepath.Join("..", "install.sh")
	body, err := os.ReadFile(installSh)
	require.NoError(t, err)
	content := string(body)

	assert.Contains(t, content, `resolve_macos_package_arch "$(uname -m)"`)
	assert.Contains(t, content, `aliyun-cli-macosx-"$VERSION"-"$MACOS_ARCH".tgz`)
	assert.NotContains(t, content, `aliyun-cli-macosx-"$VERSION"-universal.tgz`)
}

func TestResolveMacOSPackageArch_CurrentHost(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS host only")
	}
	out, err := exec.Command("uname", "-m").Output()
	require.NoError(t, err)
	machine := strings.TrimSpace(string(out))
	arch := ResolveMacOSPackageArch(machine)
	switch machine {
	case "arm64":
		assert.Equal(t, "arm64", arch)
	case "x86_64":
		assert.Equal(t, "amd64", arch)
	default:
		assert.Equal(t, "universal", arch)
	}
}
