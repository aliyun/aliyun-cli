package lib

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOSSConfigurationFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	_, opts, err := parseOSSOptions([]string{"--config-file=" + path, "--endpoint=http://localhost", "--access-key-id=id", "--access-key-secret=secret", "--sts-token=token", "--language=EN"})
	require.NoError(t, err)
	cmd := ConfigCommand{command: configCommand.command}
	require.NoError(t, cmd.Init(nil, opts))
	require.NoError(t, cmd.RunCommand())
	loaded, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "http://localhost", loaded[OptionEndpoint])
	require.Equal(t, "id", loaded[OptionAccessKeyID])
	require.Equal(t, "secret", loaded[OptionAccessKeySecret])
	require.Equal(t, "token", loaded[OptionSTSToken])
	require.Equal(t, "EN", loaded[OptionLanguage])
	_, err = LoadConfig(filepath.Dir(path))
	require.Error(t, err)
	_, opts, err = parseOSSOptions(nil)
	require.NoError(t, err)
	cmd.command.options = opts
	cmd.command.configOptions = OptionMapType{OptionLanguage: "CN"}
	cmd.rewriteAssembleOptions()
	language, err := GetString(OptionLanguage, opts)
	require.NoError(t, err)
	require.Equal(t, "CN", language)
	require.NotContains(t, cmd.command.configOptions, OptionLanguage)
	cmd.command.options = OptionMapType{}
	cmd.command.configOptions = OptionMapType{}
	cmd.rewriteAssembleOptions()
	language, err = GetString(OptionLanguage, cmd.command.options)
	require.NoError(t, err)
	require.Equal(t, OptionMap[OptionLanguage].def, language)
	cmd.command.options = OptionMapType{OptionAccessKeyID: opts[OptionAccessKeyID], OptionForce: new(bool)}
	cmd.filterNonInputOptions()
	require.Empty(t, cmd.command.options)
	require.Error(t, cmd.runCommandNonInteractive(filepath.Join(path, "child"), "EN"))
	require.NoError(t, cmd.rewriteLoadConfig(filepath.Join(path, "missing")))
	require.Empty(t, cmd.command.configOptions)
}

func TestOSSConfigurationSectionsAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	text := `[Credentials]
AccessKeyId = id
access_key_secret = secret
host = http://default
unknown = preserved
language = EN

[Default]
retry-times = 2
log-level = debug
ignored-option = ignored

[Bucket-Endpoint]
bucket = http://bucket

[Bucket-Cname]
custom = http://custom

[AkService]
ecsAk = http://credentials
`
	require.NoError(t, os.WriteFile(path, []byte(text), 0600))
	loaded, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, "id", loaded[OptionAccessKeyID])
	require.Equal(t, "secret", loaded[OptionAccessKeySecret])
	require.Equal(t, "http://default", loaded[OptionEndpoint])
	require.Equal(t, "preserved", loaded["unknown"])
	require.NotContains(t, loaded, "ignored-option")
	require.Equal(t, "http://bucket", loaded[BucketEndpointSection].(map[string]string)["bucket"])
	require.Equal(t, "http://custom", loaded[BucketCnameSection].(map[string]string)["custom"])
	require.Equal(t, "http://credentials", loaded[AkServiceSection].(map[string]string)[ItemEcsAk])
	level, err := readLoglevelFromFile(path)
	require.NoError(t, err)
	require.Equal(t, "debug", level)
	for _, text := range []string{"[Default]\nuser-agent=x\n", "[Credentials]\nlanguage=invalid\n", "[Credentials]\nretryTimes=invalid\n"} {
		require.NoError(t, os.WriteFile(path, []byte(text), 0600))
		_, err := LoadConfig(path)
		require.Error(t, err)
	}
	for _, options := range []OptionMapType{{OptionRetryTimes: "invalid"}, {OptionLanguage: "invalid"}} {
		require.Error(t, checkConfig(options))
	}
	for _, options := range []OptionMapType{{OptionRetryTimes: "2"}, {OptionLanguage: "en"}, {"unknown": "value"}} {
		require.NoError(t, checkConfig(options))
	}
	for _, group := range []map[string]configOption{CredOptionMap, DefaultOptionMap} {
		for name, option := range group {
			for _, alias := range option.showNames {
				get := getOptionNameByStr
				if _, ok := DefaultOptionMap[name]; ok {
					get = getOptionNameByDefault
				}
				got, ok := get(strings.ToUpper(alias))
				require.True(t, ok)
				require.Equal(t, name, got)
			}
		}
	}
	for _, get := range []func(string) (string, bool){getOptionNameByStr, getOptionNameByDefault} {
		name, found := get("does-not-exist")
		require.False(t, found)
		require.Empty(t, name)
	}
	require.NoError(t, os.WriteFile(path, []byte("[Credentials]\nendpoint=http://host\n"), 0600))
	level, err = readLoglevelFromFile(path)
	require.NoError(t, err)
	require.Empty(t, level)
	_, err = readLoglevelFromFile(filepath.Join(path, "missing"))
	require.Error(t, err)
	defaultPath := DefaultConfigFile
	if strings.HasPrefix(defaultPath, "~/") {
		defaultPath = filepath.Join(currentHomeDir(), strings.TrimPrefix(defaultPath, "~/"))
	}
	require.Equal(t, defaultPath, DecideConfigFile(""))
	require.Equal(t, filepath.Join(currentHomeDir(), "oss-config"), DecideConfigFile("~/oss-config"))
}

func TestOSSReporterLifecycle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reports")
	reporter, err := GetReporter(false, dir, "test")
	require.NoError(t, err)
	require.Nil(t, reporter)
	reporter.Clear()
	reporter.Comment()
	reporter.ReportError("ignored")
	reporter.Prompt(errors.New("ignored"))
	require.False(t, reporter.HasPrompt())
	reporter, err = GetReporter(true, dir, "operation details")
	require.NoError(t, err)
	require.True(t, reporter.HasPrompt())
	path := reporter.path
	reporter.Clear()
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(dir)
	require.True(t, os.IsNotExist(err))
	reporter, err = GetReporter(true, dir, "operation details")
	require.NoError(t, err)
	path = reporter.path
	reporter.ReportError("transfer failed")
	reporter.Comment()
	reporter.Prompt(errors.New("failure"))
	require.False(t, reporter.HasPrompt())
	reporter.Prompt(errors.New("failure again"))
	reporter.Clear()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "# operation details")
	require.Contains(t, string(data), "transfer failed")
	_, err = GetReporter(true, filepath.Join(path, "child"), "test")
	require.Error(t, err)
}

func TestOSSMonitorProgressStates(t *testing.T) {
	for _, scan := range []bool{true, false} {
		for _, failed := range []bool{true, false} {
			for _, exit := range []int{normalExit, errExit} {
				var monitor Monitor
				monitor.init("Restored")
				monitor.updateScanNum(5)
				monitor.updateOKNum(2)
				if failed {
					monitor.updateErrNum(1)
				}
				if scan {
					monitor.setScanEnd()
				} else {
					monitor.setScanError(errors.New("list failed"))
				}
				require.Contains(t, monitor.progressBar(false, exit), "Restored 2 objects")
				require.Contains(t, monitor.progressBar(true, exit), "Restored 2 objects")
				require.Empty(t, monitor.progressBar(true, exit))
				if scan {
					want := 40
					if failed {
						want = 60
					}
					require.Equal(t, want, monitor.getPrecent(monitor.getSnapshot()))
				} else {
					require.Zero(t, monitor.getPrecent(monitor.getSnapshot()))
				}
			}
		}
	}
	var monitor Monitor
	monitor.init("Restored")
	monitor.setScanEnd()
	require.Equal(t, 100, monitor.getPrecent(monitor.getSnapshot()))
}

func TestOSSMonitorConcurrentProgress(t *testing.T) {
	var monitor Monitor
	monitor.init("Processed")
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 1000; i++ {
			monitor.updateScanNum(2)
			monitor.updateOKNum(1)
			monitor.updateErrNum(1)
		}
		monitor.setScanEnd()
	}()
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 1000; i++ {
			monitor.progressBar(false, normalExit)
			snapshot := monitor.getSnapshot()
			if snapshot.okNum > 1000 || snapshot.errNum > 1000 {
				t.Error("progress counters exceeded the number of processed objects")
			}
		}
	}()
	close(start)
	workers.Wait()
	snapshot := monitor.getSnapshot()
	require.Equal(t, int64(2000), monitor.totalNum)
	require.Equal(t, int64(1000), snapshot.okNum)
	require.Equal(t, int64(1000), snapshot.errNum)
	require.Equal(t, int64(2000), snapshot.dealNum)
	final := monitor.progressBar(true, normalExit)
	require.Contains(t, final, "Total 2000 objects")
	require.Contains(t, final, "Processed 1000 objects")
	require.Contains(t, final, "Error 1000 objects")
}

func TestOSSConfigurationInteractiveDefaultsAndErrors(t *testing.T) {
	for _, language := range []string{"EN", "CH"} {
		path := filepath.Join(t.TempDir(), "config")
		cmd := ConfigCommand{command: configCommand.command}
		withBucketInput(t, "invalid\n", func() { require.ErrorContains(t, cmd.configInteractive(path, language), "invalid option value") })
		withBucketInput(t, "", func() { require.NoError(t, cmd.configInteractive(path, language)) })
		loaded, err := LoadConfig(path)
		require.NoError(t, err)
		require.Equal(t, language, loaded[OptionLanguage])
		withBucketInput(t, "", func() { require.NoError(t, cmd.configInteractive(path, language)) })
		withBucketInput(t, "", func() { require.Error(t, cmd.configInteractive(filepath.Join(path, "child"), language)) })
		withBucketInput(t, "", func() { require.ErrorContains(t, cmd.runCommandInteractive(path, language), "interactive terminal") })
	}
}

func TestOSSCopyAndRemoveMonitorStates(t *testing.T) {
	for _, op := range []operationType{operationTypePut, operationTypeGet, operationTypeCopy} {
		for _, scanned := range []bool{false, true} {
			for _, failed := range []bool{false, true} {
				var m CPMonitor
				m.init(op)
				m.updateScanSizeNum(100, 5)
				m.updateScanNum(1)
				m.updateFile(10, 1)
				m.updateDir(0, 1)
				m.updateSkip(5, 1)
				m.updateSkipDir(1)
				m.updateDealSize(2)
				m.updateTransferSize(3)
				if scanned {
					m.setScanEnd()
				}
				if failed {
					m.updateErr(4, 1)
					m.setScanError(errors.New("scan failed"))
				}
				snap := m.getSnapshot()
				require.Equal(t, int64(17), snap.dealSize)
				require.Equal(t, int64(3), snap.okNum)
				require.Contains(t, m.getDealNumDetail(snap), "skip 1")
				require.NotEmpty(t, m.getOKNumDetail(snap))
				require.Empty(t, m.getNumDetail(&CPMonitorSnap{}, false))
				require.Empty(t, m.getNumDetail(&CPMonitorSnap{}, true))
				require.NotEmpty(t, m.getNumDetail(&CPMonitorSnap{dirNum: 1, okNum: 1}, true))
				require.NotEmpty(t, m.getSizeDetail(&CPMonitorSnap{skipSize: 3}))
				require.NotEmpty(t, m.getSizeDetail(&CPMonitorSnap{transferSize: 3}))
				require.Empty(t, m.getSkipSize(&CPMonitorSnap{}))
				require.Equal(t, 2.0, m.getSpeed(&CPMonitorSnap{incrementSize: 2048, duration: int64(time.Second)}))
				require.Empty(t, m.progressBar(false, normalExit))
				m.tickDuration = 0
				require.Contains(t, m.progressBar(false, normalExit), "Dealed num:")
				require.NotEmpty(t, m.getFinishBar(normalExit))
				require.NotEmpty(t, m.getFinishBar(errExit))
				require.NotEmpty(t, m.progressBar(true, normalExit))
				require.Empty(t, m.progressBar(true, normalExit))
			}
		}
	}
	var cp CPMonitor
	cp.init(operationTypeGet)
	cp.setScanEnd()
	require.Equal(t, 100.0, cp.getPrecent(cp.getSnapshot()))
	cp.updateScanNum(2)
	require.Zero(t, cp.getPrecent(cp.getSnapshot()))
	for _, op := range []int64{0, objectType, multipartType, allType, allType | bucketType} {
		for _, scanned := range []bool{false, true} {
			var m RMMonitor
			m.init()
			m.setOP(op)
			m.updateOP(op)
			m.updateScanNum(4)
			m.updateScanUploadIdNum(4)
			m.updateObjectNum(1)
			m.updateUploadIdNum(1)
			m.updateErrObjectNum(1)
			m.updateErrUploadIdNum(1)
			if scanned {
				m.setScanEnd()
			} else {
				m.setScanError(errors.New("list failed"))
			}
			require.NotEmpty(t, m.progressBar(false, normalExit))
			require.NotEmpty(t, m.getFinishBar(errExit))
			m.updateRemovedBucket("bucket")
			require.Equal(t, "bucket", m.getSnapshot().removedBucket)
			require.NotEmpty(t, m.progressBar(true, normalExit))
			require.Empty(t, m.progressBar(true, normalExit))
		}
	}
	var rm RMMonitor
	rm.init()
	rm.setOP(objectType)
	rm.setScanEnd()
	require.Equal(t, 100, rm.getPrecent(rm.getSnapshot()))
	require.Equal(t, normalExit, rm.getExitStat(rm.getSnapshot(), normalExit))
	rm.setOP(bucketType)
	require.Equal(t, errExit, rm.getExitStat(rm.getSnapshot(), normalExit))
}
