package lib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestLegacyOptionFactoriesAndTypeErrors(t *testing.T) {
	options := initOption()
	require.Len(t, options, len(OptionMap))
	for name, value := range options {
		switch OptionMap[name].optionType {
		case OptionTypeFlagTrue:
			require.IsType(t, new(bool), value)
		case OptionTypeStrings:
			require.IsType(t, new([]string), value)
		default:
			require.IsType(t, new(string), value)
		}
	}
	for _, option := range []Option{{name: "-x"}, {nameAlias: "--example"}, {name: "-x", nameAlias: "--example"}} {
		names, err := makeNames(option)
		require.NoError(t, err)
		require.NotEmpty(t, names)
	}
	_, err := makeNames(Option{})
	require.Error(t, err)
	_, err = stringOption(Option{})
	require.Error(t, err)
	_, err = stringsOption(Option{})
	require.Error(t, err)
	_, err = flagTrueOption(Option{})
	require.Error(t, err)
	_, err = GetInt(OptionRoutines, OptionMapType{OptionRoutines: boolPointer(true)})
	require.Error(t, err)
	_, err = GetInt(OptionRoutines, nil)
	require.Error(t, err)
	_, err = GetStrings(OptionQueryParam, OptionMapType{OptionQueryParam: stringPointer("value")})
	require.Error(t, err)
	_, err = GetStrings(OptionQueryParam, nil)
	require.Error(t, err)
	_, err = GetBool(OptionForce, OptionMapType{OptionForce: stringPointer("yes")})
	require.Error(t, err)
	require.Error(t, checkOption(OptionMapType{OptionRoutines: stringPointer("100000")}))
}

func TestEveryCommandRejectsInvalidInvocation(t *testing.T) {
	previousCommand, previousSync := copyCommand.command, copyCommand.cpOption.bSyncCommand
	t.Cleanup(func() {
		copyCommand.command, copyCommand.cpOption.bSyncCommand = previousCommand, previousSync
	})
	for _, original := range GetAllCommands() {
		value := reflect.ValueOf(original).Elem()
		metadata := value.FieldByName("command")
		name := metadata.FieldByName("name").String()
		t.Run(name, func(t *testing.T) {
			copy := reflect.New(value.Type())
			copy.Elem().Set(value)
			var args []string
			var options OptionMapType
			want := "argument"
			if metadata.FieldByName("minArgc").Int() == 0 {
				maximum := metadata.FieldByName("maxArgc").Int()
				if maximum > 1024 {
					options = OptionMapType{OptionConfigFile: stringPointer(filepath.Join(t.TempDir(), "missing"))}
					want = "config file"
				} else {
					args = make([]string, int(maximum)+1)
				}
			}
			err := copy.Interface().(Commander).Init(args, options)
			require.ErrorContains(t, err, want)
		})
	}
}

func TestMetadataPromptsAndHeaderOptionTypes(t *testing.T) {
	for _, language := range []string{"en", "ch"} {
		cmd := SetMetaCommand{command: setMetaCommand.command}
		cmd.command.args = []string{"oss://bucket/key"}
		for _, tc := range []struct {
			input, want string
			failed      bool
		}{{"y\n", "", false}, {"n\nContent-Type:text/plain\n", "Content-Type:text/plain", false}, {"n\n", "", true}, {"other\n", "", true}} {
			withBucketInput(t, tc.input, func() {
				got, err := cmd.getMetaData(false, language)
				if tc.failed {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.want, got)
				}
			})
		}
		got, err := cmd.getMetaData(true, language)
		require.NoError(t, err)
		require.Empty(t, got)
		cloud := CloudURL{bucket: "bucket"}
		withBucketInput(t, "n\n", func() { require.NoError(t, cmd.checkOptions(cloud, true, false, false, true, language, "", "")) })
		withBucketInput(t, "y\ny\n", func() { require.NoError(t, cmd.checkOptions(cloud, false, false, false, true, language, "", "")) })
		withBucketInput(t, "y\nn\n", func() { require.Error(t, cmd.checkOptions(cloud, false, false, false, true, language, "", "")) })
	}
	for _, tc := range []struct{ name, value string }{{"Content-Type", "text/plain"}, {"x-oss-object-acl", "private"}, {"Expires", "Thu, 01 Oct 2026 00:00:00 GMT"}, {"Expires", "2026-10-01T00:00:00Z"}, {"x-oss-storage-class", "Archive"}} {
		option, err := getOSSOption(headerOptionMap, tc.name, tc.value)
		require.NoError(t, err)
		require.NotNil(t, option)
	}
	_, err := getOSSOption(headerOptionMap, "Expires", "invalid")
	require.Error(t, err)
	_, err = getOSSOption(map[string]interface{}{"invalid": 1}, "invalid", "value")
	require.Error(t, err)
	cmd := Command{}
	_, err = cmd.parseHeaders("x-oss-meta-team:value", true)
	require.Error(t, err)
	headers, err := cmd.parseHeaders("", false)
	require.NoError(t, err)
	require.Empty(t, headers)
	opts, err := cmd.getOSSOptions(headerOptionMap, map[string]string{"x-oss-custom": "value"})
	require.NoError(t, err)
	require.Len(t, opts, 1)
	_, err = cmd.getOSSTagging("a=b&&c=d")
	require.Error(t, err)
}

func TestRemoveFlagValidationAndQuietDeletes(t *testing.T) {
	rc := RemoveCommand{command: Command{args: []string{"oss://bucket"}}}
	for _, tc := range []struct {
		object                 string
		multipart, all, bucket bool
	}{
		{"", false, false, false}, {"", true, false, true}, {"key", true, false, true}, {"key", false, false, true},
	} {
		require.Error(t, rc.checkOption(CloudURL{bucket: "bucket", object: tc.object}, tc.multipart, tc.all, tc.bucket))
	}
	rc.rmOption.recursive, rc.rmOption.versionId = true, "version"
	require.Error(t, rc.checkOption(CloudURL{bucket: "bucket", object: "key"}, false, false, false))
	rc.rmOption.recursive, rc.rmOption.allVersions = false, true
	require.Error(t, rc.checkOption(CloudURL{bucket: "bucket", object: "key"}, false, false, false))
	rc.rmOption.versionId = ""
	for _, input := range []string{"n\n", "y\n"} {
		withBucketInput(t, input, func() { require.Equal(t, input == "y\n", rc.confirmRemoveBucket(CloudURL{bucket: "bucket"})) })
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		calls.Add(1)
		io.WriteString(w, `<DeleteResult/>`)
	}))
	defer server.Close()
	rc.command = localOSSCommand(t, server.URL)
	bucket, err := rc.command.ossBucket("bucket")
	require.NoError(t, err)
	deleted, err := rc.ossBatchDeleteObjectsRetry(bucket, []string{"first", "second"})
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
	require.EqualValues(t, 1, calls.Load())
	deleted, err = rc.ossBatchDeleteObjectsRetryVersion(bucket, []oss.DeleteObject{{Key: "first", VersionId: "v1"}, {Key: "second", VersionId: "v2"}})
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
	require.EqualValues(t, 2, calls.Load())
}

func TestSyncDeleteBatchesAndConfirmation(t *testing.T) {
	server, writes := newDataServer(t)
	defer server.Close()
	sc := SyncCommand{command: localOSSCommand(t, server.URL)}
	keys := map[string]string{}
	for i := 0; i <= MaxBatchCount; i++ {
		keys[fmt.Sprint(i)] = "prefix/"
	}
	sc.syncOption.force = true
	captureBucketOutput(t, func() { require.NoError(t, sc.DeleteExtraObjects(keys, CloudURL{bucket: "bucket"})) })
	require.EqualValues(t, 2, writes.Load())
	sc.syncOption.force = false
	for _, input := range []string{"n\n", "y\n"} {
		withBucketInput(t, input, func() { require.Equal(t, input == "y\n", sc.confirm([]string{"first", "second"})) })
	}
	for _, tc := range []struct {
		source, dest StorageURLer
		want         operationType
	}{
		{CloudURL{}, FileURL{}, operationTypeGet}, {CloudURL{}, CloudURL{}, operationTypeCopy}, {FileURL{}, CloudURL{}, operationTypePut},
	} {
		require.Equal(t, tc.want, sc.getCommandType(tc.source, tc.dest))
	}
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer denied.Close()
	sc.command = localOSSCommand(t, denied.URL)
	bucket, err := sc.command.ossBucket("bucket")
	require.NoError(t, err)
	require.Error(t, sc.BatchRmObjects(bucket, []string{"first"}, nil))
	partial := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<DeleteResult><Deleted><Key>first</Key></Deleted></DeleteResult>`)
	}))
	defer partial.Close()
	sc.command = localOSSCommand(t, partial.URL)
	bucket, err = sc.command.ossBucket("bucket")
	require.NoError(t, err)
	require.ErrorContains(t, sc.BatchRmObjects(bucket, []string{"first"}, nil), "first")
}

func TestAppendAndCopyProgressFailureEvents(t *testing.T) {
	previousProgress := chProgressSignal
	chProgressSignal = make(chan chProgressSignalType, 10)
	t.Cleanup(func() { chProgressSignal = previousProgress })
	appendListener := AppendProgressListener{}
	appendListener.ProgressChanged(&oss.ProgressEvent{EventType: oss.TransferDataEvent, ConsumedBytes: 1, TotalBytes: 4})
	appendListener.lastMilliSecond = time.Now().Add(-2 * time.Second).UnixMilli()
	captureBucketOutput(t, func() {
		appendListener.ProgressChanged(&oss.ProgressEvent{EventType: oss.TransferCompletedEvent, ConsumedBytes: 4, TotalBytes: 4})
	})
	require.Equal(t, int64(4), appendListener.currSize)
	var monitor CPMonitor
	monitor.init(operationTypePut)
	listener := OssProgressListener{monitor: &monitor}
	listener.ProgressChanged(&oss.ProgressEvent{EventType: oss.TransferDataEvent, RwBytes: 4})
	listener.ProgressChanged(&oss.ProgressEvent{EventType: oss.TransferFailedEvent, ConsumedBytes: 4})
	require.True(t, listener.failedEvent)
	require.Zero(t, monitor.getSnapshot().dealSize)
}

func TestPasswordRejectsUnavailableInteractiveInput(t *testing.T) {
	for _, prompt := range []string{"", "Password: "} {
		withBucketInput(t, "", func() {
			password, err := GetPassword(prompt)
			require.Error(t, err)
			require.Empty(t, password)
		})
	}
	previous := activeMachine
	t.Cleanup(func() { activeMachine = previous })
	activeMachine = &machineInvocation{nonInteractive: true}
	password, err := GetPassword("Password: ")
	require.ErrorIs(t, err, errConfirmationRequired)
	require.Empty(t, password)
	require.True(t, activeMachine.confirmation.Load())
}
