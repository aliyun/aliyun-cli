//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetryConfigureSetRecordsTargetAuthenticationMode(t *testing.T) {
	const currentName = "telemetry-private-current-profile"
	const otherName = "telemetry-private-target-profile"
	const accessKeyID = "telemetry-private-fixture-access-key-id"
	const accessKeySecret = "telemetry-private-fixture-access-key-secret"
	const token = "telemetry-private-fixture-sts-token"
	const initialRegion = "cn-telemetry-private-initial"
	const updatedRegion = "cn-telemetry-private-updated"

	for _, test := range []struct {
		name              string
		currentMode       config.AuthenticateMode
		targetMode        config.AuthenticateMode
		sameCurrent       bool
		newProfile        bool
		modeFlag          string
		importedMode      config.AuthenticateMode
		validationFailure bool
		wantMode          string
		wantSource        string
	}{
		{name: "other_existing_ak", currentMode: config.Anonymous, targetMode: config.AK, wantMode: "AK", wantSource: telemetry.AuthModeSourceExisting},
		{name: "other_existing_anonymous", currentMode: config.AK, targetMode: config.Anonymous, wantMode: "Anonymous", wantSource: telemetry.AuthModeSourceExisting},
		{name: "same_current_no_override", currentMode: config.AK, sameCurrent: true, wantMode: "AK", wantSource: telemetry.AuthModeSourceExisting},
		{name: "new_profile_defaults_to_ak", currentMode: config.Anonymous, newProfile: true, wantMode: "AK", wantSource: telemetry.AuthModeSourceDefault},
		{name: "empty_existing_mode_defaults_to_ak", currentMode: config.Anonymous, wantMode: "AK", wantSource: telemetry.AuthModeSourceDefault},
		{name: "explicit_ak_case_normalized", currentMode: config.Anonymous, targetMode: config.Anonymous, modeFlag: "aK", wantMode: "AK", wantSource: telemetry.AuthModeSourceExplicit},
		{name: "existing_target_validation_failure", currentMode: config.Anonymous, targetMode: config.AK, validationFailure: true, wantMode: "AK", wantSource: telemetry.AuthModeSourceExisting},
		{name: "imported_existing_sts", currentMode: config.Anonymous, targetMode: config.AK, importedMode: config.StsToken, wantMode: "StsToken", wantSource: telemetry.AuthModeSourceExisting},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			fixtureProfile := func(name string, mode config.AuthenticateMode) config.Profile {
				return config.Profile{
					Name: name, Mode: mode, RegionId: initialRegion, Language: "en",
					AccessKeyId: accessKeyID, AccessKeySecret: accessKeySecret, StsToken: token,
				}
			}
			initial := config.Configuration{
				CurrentProfile: currentName,
				Profiles:       []config.Profile{fixtureProfile(currentName, test.currentMode)},
			}
			targetName := otherName
			args := []string{"configure", "set"}
			wantRegion := updatedRegion
			if test.sameCurrent {
				targetName = currentName
				wantRegion = initialRegion
			} else {
				args = append(args, "--profile", targetName, "--region", updatedRegion)
				if !test.newProfile {
					target := fixtureProfile(targetName, test.targetMode)
					if test.validationFailure {
						target.AccessKeySecret = ""
					}
					initial.Profiles = append(initial.Profiles, target)
				}
			}
			if test.newProfile {
				args = append(args, "--access-key-id", accessKeyID, "--access-key-secret", accessKeySecret)
			}
			if test.modeFlag != "" {
				args = append(args, "--mode", test.modeFlag)
			}

			defaultPath := filepath.Join(dir, "config.json")
			initialBytes, err := json.Marshal(initial)
			require.NoError(t, err)
			initialFiles := map[string][]byte{defaultPath: initialBytes}
			persistedPath := defaultPath
			if test.importedMode != "" {
				persistedPath = filepath.Join(home, "telemetry-private-import-config.json")
				imported := config.Configuration{
					CurrentProfile: targetName,
					Profiles:       []config.Profile{fixtureProfile(targetName, test.importedMode)},
				}
				initialFiles[persistedPath], err = json.Marshal(imported)
				require.NoError(t, err)
				args = append(args, "--config-path", persistedPath)
			}
			writeInitialFiles := func() {
				for path, data := range initialFiles {
					require.NoError(t, os.WriteFile(path, data, 0600))
				}
			}
			readPersistedFiles := func() map[string]string {
				files := make(map[string]string, len(initialFiles))
				for path := range initialFiles {
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					files[path] = string(data)
				}
				return files
			}

			writeInitialFiles()
			baseline := runTelemetryPermissionCommand(t, home, true, args)
			if test.validationFailure {
				require.Greater(t, baseline.exitCode, 0)
				require.Contains(t, baseline.stderr, "access_key_secret is not configured")
			} else {
				require.Zero(t, baseline.exitCode, baseline.stderr)
			}
			baselineFiles := readPersistedFiles()
			require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
			require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))

			// configure set selects the target as current when it saves. Restore
			// both inputs so the enabled invocation starts with the same current
			// profile, rather than accidentally hiding a stale-current-mode bug.
			writeInitialFiles()
			require.Equal(t, string(initialBytes), readPersistedFiles()[defaultPath])
			got := runTelemetryPermissionCommand(t, home, false, args)
			require.Equal(t, baseline, got, "telemetry must preserve configure output and exit status")
			gotFiles := readPersistedFiles()
			require.Equal(t, baselineFiles, gotFiles, "telemetry must preserve the persisted configuration")

			var saved config.Configuration
			require.NoError(t, json.Unmarshal([]byte(gotFiles[persistedPath]), &saved))
			if test.validationFailure {
				require.Equal(t, string(initialBytes), gotFiles[defaultPath], "validation failure must not save any configuration")
				require.Equal(t, currentName, saved.CurrentProfile)
				wantRegion = initialRegion
			} else {
				require.Equal(t, targetName, saved.CurrentProfile)
			}
			target, exists := saved.GetProfile(targetName)
			require.True(t, exists)
			require.Equal(t, config.AuthenticateMode(test.wantMode), target.Mode)
			require.Equal(t, wantRegion, target.RegionId)
			if test.importedMode != "" {
				require.Equal(t, string(initialBytes), gotFiles[defaultPath], "import must retain its existing save destination")
				require.Equal(t, token, target.StsToken)
			}
			if target.Mode == config.AK && !test.validationFailure {
				require.Equal(t, accessKeyID, target.AccessKeyId)
				require.Equal(t, accessKeySecret, target.AccessKeySecret)
			}

			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, "configure set", event.Command)
			require.Equal(t, telemetry.CommandTypeCore, event.CommandType)
			if test.validationFailure {
				require.NotEqual(t, "Success", event.Result)
			} else {
				require.Equal(t, "Success", event.Result)
			}
			require.Equal(t, test.wantMode, event.AuthMode)
			require.Equal(t, test.wantSource, event.AuthModeSource)
			require.Contains(t, []string{"AK", "Anonymous", "StsToken"}, event.AuthMode)
			require.Contains(t, []string{telemetry.AuthModeSourceExisting, telemetry.AuthModeSourceDefault, telemetry.AuthModeSourceExplicit}, event.AuthModeSource)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{currentName, otherName, accessKeyID, accessKeySecret, token, initialRegion, updatedRegion, persistedPath, home} {
				require.NotContains(t, string(data), private)
			}
		})
	}
}
