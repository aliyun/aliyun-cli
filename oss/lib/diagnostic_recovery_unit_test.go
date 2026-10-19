package lib

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestCommandManagerVersionHelpAndLogConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	previousArgs, previousLine, previousHelp := os.Args, commandLine, helpCommand.command
	previousLevel, previousName, previousLogger, previousFile := logLevel, logName, utilLogger, logFile
	t.Cleanup(func() {
		UnInitLogger()
		os.Args, commandLine, helpCommand.command = previousArgs, previousLine, previousHelp
		logLevel, logName, utilLogger, logFile = previousLevel, previousName, previousLogger, previousFile
	})
	logFile, utilLogger = nil, nil
	executable := filepath.Join(t.TempDir(), "ossutil.exe")
	for _, level := range []string{"info", "debug"} {
		os.Args = []string{executable, "--version", "--loglevel=" + level}
		output := captureBucketOutput(t, func() { require.NoError(t, ParseAndRunCommand()) })
		require.Contains(t, output, "ossutil version:")
		data, err := os.ReadFile(filepath.Join(filepath.Dir(executable), logName))
		require.NoError(t, err)
		require.Contains(t, string(data), "ossutil run begin")
	}
	os.Args = []string{executable, "--loglevel=invalid"}
	require.ErrorContains(t, ParseAndRunCommand(), "loglevel must be")
	os.Args = []string{executable, "--retry-times=invalid"}
	require.ErrorContains(t, ParseAndRunCommand(), "not int64")
	_, options, err := parseOSSOptions(nil, "help")
	require.NoError(t, err)
	base := Command{options: options}
	output := captureBucketOutput(t, func() {
		show, err := RunCommand(nil, base.options)
		require.NoError(t, err)
		require.False(t, show)
	})
	require.Contains(t, output, "Commands:")
	manager := CommandManager{}
	manager.Init()
	_, err = manager.RunCommand("not-registered", nil, base.options)
	require.ErrorContains(t, err, "no such command")
	_, err = manager.RunCommand("help", []string{"missing", "extra"}, base.options)
	require.ErrorContains(t, err, "argument")
	_, err = getLoglevelFromOptions(nil)
	require.Error(t, err)
	level, err := getLoglevelFromOptions(OptionMapType{OptionLogLevel: stringPointer("info")})
	require.NoError(t, err)
	require.Equal(t, "info", level)
	hc := HelpCommand{command: helpCommand.command}
	require.NoError(t, hc.rewriteLoadConfig(filepath.Join(t.TempDir(), "missing")))
	require.Empty(t, hc.command.configOptions)
	hc.command.args = []string{"cp"}
	output = captureBucketOutput(t, func() { require.NoError(t, hc.RunCommand()) })
	require.Contains(t, output, "SYNOPSIS")
	hc.command.args = []string{"missing"}
	require.ErrorContains(t, hc.RunCommand(), "missing")
}

func TestUpdateTemporaryPathsAndConfirmedReplacement(t *testing.T) {
	t.Chdir(t.TempDir())
	previousArgs, previousClient, previousVersion := os.Args, http.DefaultClient, vVersion
	t.Cleanup(func() { os.Args, http.DefaultClient, vVersion = previousArgs, previousClient, previousVersion })
	path := filepath.Join(t.TempDir(), "ossutil.exe")
	os.Args = []string{path}
	uc := UpdateCommand{command: localOSSCommand(t, "unused")}
	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "config")} {
		if filepath.Base(path) == "config" {
			require.NoError(t, os.WriteFile(path, []byte("[Credentials]\nendpoint=http://localhost\n"), 0600))
		}
		require.NoError(t, uc.rewriteLoadConfig(path))
		if filepath.Base(path) == "config" {
			require.Equal(t, "http://localhost", uc.command.configOptions[OptionEndpoint])
		} else {
			require.Empty(t, uc.command.configOptions)
		}
	}
	var requests atomic.Int32
	http.DefaultClient = &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		body := "replacement"
		if r.URL.Path == "/"+updateVersionObject {
			body = "9.0.0\n"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	require.Error(t, uc.updateVersion("9.0.0", "en"))
	require.NoError(t, os.WriteFile(path, []byte("original"), 0700))
	_, renamed := getBinaryPath()
	require.NoError(t, os.Mkdir(renamed, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(renamed, "child"), nil, 0600))
	require.ErrorContains(t, uc.updateVersion("9.0.0", "en"), "update binary error")
	require.NoError(t, os.RemoveAll(renamed))
	vVersion = "1.0.0"
	uc.command.options[OptionForce] = boolPointer(false)
	uc.command.options[OptionLanguage] = stringPointer("CN")
	withBucketInput(t, "y\n", func() { require.NoError(t, uc.RunCommand()) })
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "replacement", string(data))
	require.GreaterOrEqual(t, requests.Load(), int32(2))
	require.NoError(t, os.WriteFile(renamed, []byte("restore"), 0700))
	require.NoError(t, uc.revertRename(path, renamed))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "restore", string(data))
	require.ErrorIs(t, uc.revertRename(path, renamed), os.ErrNotExist)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "download") }))
	defer server.Close()
	http.DefaultClient = server.Client()
	var pathError *os.PathError
	require.ErrorAs(t, uc.ossAnonymousGetToFile(server.URL, filepath.Join(path, "child")), &pathError)
	require.Equal(t, "open", pathError.Op)
}

func TestProbeFailuresPreserveFilesAndWriteDiagnostics(t *testing.T) {
	t.Chdir(t.TempDir())
	var headCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && headCount.Add(1) == 1 {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>local rejection</Message></Error>`)
	}))
	defer server.Close()
	logPath := filepath.Join(t.TempDir(), "probe.log")
	log, err := os.Create(logPath)
	require.NoError(t, err)
	defer log.Close()
	pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
	pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "key"
	pc.pbOption.disableNetDetect, pc.pbOption.logFile, pc.pbOption.logName = true, log, logPath
	pc.command.args = []string{filepath.Join(t.TempDir(), "upload")}
	require.NoError(t, os.WriteFile(pc.command.args[0], []byte("source"), 0600))
	require.ErrorContains(t, pc.probeUpload(), "AccessDenied")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "upload file:failure")
	require.Contains(t, string(data), "AccessDenied")
	require.FileExists(t, pc.command.args[0])
	for _, upload := range []func(string, string) error{pc.probeUploadFileAppend, pc.probeUploadFileMultiPart, pc.probeUploadFileNormal} {
		missing := filepath.Join(t.TempDir(), "missing")
		require.ErrorContains(t, upload(missing, "key"), "missing")
	}
	existing := filepath.Join(t.TempDir(), "existing")
	require.NoError(t, os.WriteFile(existing, []byte("original"), 0600))
	pc.command.args = []string{existing}
	withBucketInput(t, "n\n", func() { require.NoError(t, pc.probeDownloadObject(CloudURL{bucket: "bucket", object: "key"}, false)) })
	data, err = os.ReadFile(existing)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	withBucketInput(t, "y\n", func() {
		require.ErrorContains(t, pc.probeDownloadObject(CloudURL{bucket: "bucket", object: "key"}, false), "GetObjectToFile")
	})
	pc.command.args = []string{filepath.Join(existing, "child")}
	require.Error(t, pc.probeDownloadObject(CloudURL{bucket: "bucket", object: "key"}, false))
}

func TestCommonObjectListRejectsSymlinkMetadataFailures(t *testing.T) {
	for _, missingLength := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				if !missingLength {
					w.WriteHeader(403)
				}
				return
			}
			io.WriteString(w, `<ListBucketResult><Contents><Key>link</Key><Type>Symlink</Type><Size>1</Size></Contents></ListBucketResult>`)
		}))
		base := localOSSCommand(t, server.URL)
		bucket, err := base.ossBucket("bucket")
		require.NoError(t, err)
		objects := make(chan objectInfoType, 1)
		require.Error(t, getObjectListCommon(bucket, CloudURL{bucket: "bucket"}, objects, false, nil, nil))
		_, open := <-objects
		require.False(t, open)
		server.Close()
	}
	base := Command{}
	options, err := AddStringsToOption(nil, nil)
	require.NoError(t, err)
	require.Empty(t, options)
	var monitor Monitor
	monitor.init("Processed")
	base.updateMonitor(errors.New("failure"), &monitor)
	require.EqualValues(t, 1, monitor.getSnapshot().errNum)
	require.True(t, doesSingleObjectMatchPatterns("key", nil))
	require.Equal(t, []string{"key"}, matchFiltersForStrsInArray([]string{"key"}, nil))
}

func TestBridgeCredentialEndpointAndHelpFailuresStayLocal(t *testing.T) {
	clearEndpointTestEnv(t)
	for _, scenario := range []string{"sts snapshot", "external disabled", "credentials URI", "resolved retry limit", "region", "missing config"} {
		t.Run(scenario, func(t *testing.T) {
			clearEndpointTestEnv(t)
			profile := config.Profile{Name: "default", Mode: config.StsToken, AccessKeyId: "id", AccessKeySecret: "secret", StsToken: "token", RegionId: "cn-hangzhou", Language: "en"}
			args := []string{"oss://bucket"}
			switch scenario {
			case "external disabled":
				profile.Mode, profile.ProcessCommand = config.External, "unused"
				t.Setenv(config.EnvDisableExternalProcess, "true")
			case "credentials URI":
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
				defer server.Close()
				profile.Mode, profile.CredentialsURI = config.CredentialsURI, server.URL
			case "resolved retry limit":
				profile.RetryCount = int(MaxRetryTimes + 1)
			case "region":
				args = append(args, "--region=bad region")
			}
			data, err := json.Marshal(map[string]interface{}{"current": "default", "profiles": []config.Profile{profile}})
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "config.json")
			if scenario != "missing config" {
				require.NoError(t, os.WriteFile(path, data, 0600))
			}
			ctx := bridgeTestContext(NewCommandBridge(listCommand.command))
			config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
			config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
			previous := parseAndRunCommandImpl
			defer func() { parseAndRunCommandImpl = previous }()
			calls := 0
			parseAndRunCommandImpl = func() error {
				calls++
				mode, err := GetString(OptionMode, bridgeResolvedOptions)
				require.NoError(t, err)
				require.Equal(t, "StsToken", mode)
				return nil
			}
			err = parseAndRunCommandFromCli(ctx, args, &listCommand.command)
			if scenario == "sts snapshot" {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
			} else {
				require.Error(t, err)
				require.Zero(t, calls)
			}
		})
	}
	ctx := bridgeTestContext(NewCommandBridge(copyCommand.command))
	positional, help, err := parseBridgeFlags(ctx, []string{"source", "-h"})
	require.NoError(t, err)
	require.True(t, help)
	require.Equal(t, []string{"source"}, positional)
	_, _, err = parseBridgeFlags(ctx, []string{"--recursive=true"})
	require.ErrorContains(t, err, "does not take a value")
	require.Equal(t, []string{"oss://bucket"}, ossPositionalArgs(ctx, []string{"-e", "http://local.test", "oss://bucket"}))
	config.RegionFlag(ctx.Flags()).SetAssigned(true)
	config.RegionFlag(ctx.Flags()).SetValue("cn-beijing")
	endpoint, err := resolveOSSEndpoint(ctx, nil, config.Profile{RegionId: "cn-hangzhou"})
	require.NoError(t, err)
	require.Equal(t, "oss-cn-beijing.aliyuncs.com", endpoint)
	ctx = bridgeTestContext(NewCommandBridge(updateCommand.command))
	require.ErrorContains(t, parseAndRunCommandFromCli(ctx, nil, &updateCommand.command), "not supported in the host CLI")
}

func TestOSSClientRejectsIncompleteModesAndPropagatesSTSFailures(t *testing.T) {
	for _, mode := range []string{"AK", "StsToken", "RamRoleArn"} {
		cmd := localOSSCommand(t, "http://oss.test")
		cmd.options[OptionMode], cmd.options[OptionAccessKeyID] = &mode, stringPointer("")
		_, err := cmd.ossClient("bucket")
		require.ErrorContains(t, err, "accessKeyID is empty")
	}
	cmd := localOSSCommand(t, "http://[::1")
	_, err := cmd.ossClient("bucket")
	require.Error(t, err)
	previousHost, previousLevel := StsHost, logLevel
	t.Cleanup(func() { StsHost, logLevel = previousHost, previousLevel })
	cmd = localOSSCommand(t, "http://oss.test")
	cmd.options[OptionMode], cmd.options[OptionRamRoleArn], cmd.options[OptionSTSRegion] = stringPointer("RamRoleArn"), stringPointer("role"), stringPointer("cn-test%zz")
	_, err = cmd.ossClient("bucket")
	require.ErrorContains(t, err, "invalid URL escape")
	require.Equal(t, "https://sts.cn-test%zz.aliyuncs.com", StsHost)
	cmd = localOSSCommand(t, "http://oss.test")
	logLevel = oss.Info
	client, err := cmd.ossClient("bucket")
	require.NoError(t, err)
	require.Equal(t, oss.Info, client.Config.LogLevel)
}

func TestInteractiveConfigurationPersistsEnteredLanguageAndCredential(t *testing.T) {
	ending := "\n"
	if runtime.GOOS == "windows" {
		ending = "\r\n"
	}
	for _, language := range []string{"en", "ch"} {
		path := filepath.Join(t.TempDir(), "config")
		cmd := ConfigCommand{}
		withBucketInput(t, "EN"+ending+"entered-value"+ending, func() { require.NoError(t, cmd.configInteractive(path, language)) })
		loaded, err := LoadConfig(path)
		require.NoError(t, err)
		require.Equal(t, "EN", loaded[OptionLanguage])
		entered := 0
		for _, name := range []string{OptionEndpoint, OptionAccessKeyID, OptionAccessKeySecret, OptionSTSToken} {
			if loaded[name] == "entered-value" {
				entered++
			}
		}
		require.Equal(t, 1, entered)
	}
}
