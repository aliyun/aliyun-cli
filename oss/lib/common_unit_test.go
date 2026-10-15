package lib

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"hash/fnv"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestCommandHelpForEveryRegisteredCommand(t *testing.T) {
	old := helpCommand.command.options
	t.Cleanup(func() { helpCommand.command.options = old })
	for _, language := range []string{LEnglishLanguage, "CH"} {
		helpCommand.command.options = OptionMapType{OptionLanguage: &language}
		hc := HelpCommand{command: helpCommand.command}
		groups, commands := hc.getCommandMap()
		require.Contains(t, hc.formatWholeHelp(groups), hc.getUsageText())
		require.Empty(t, hc.formatWholeHelp(nil))
		for name, command := range commands {
			t.Run(language+"/"+name, func(t *testing.T) {
				hc.command.args = []string{name}
				help, err := hc.formatCommandHelp(commands)
				require.NoError(t, err)
				require.Contains(t, help, "SYNOPSIS")
				cmd := reflect.ValueOf(command).Elem().FieldByName("command")
				spec := cmd.FieldByName("specChinese")
				if language == LEnglishLanguage {
					spec = cmd.FieldByName("specEnglish")
				}
				require.Contains(t, help, strings.TrimSpace(spec.FieldByName("synopsisText").String()))
				require.Contains(t, command.(FormatHelper).formatHelpForWhole(), name)
			})
		}
		hc.command.args = []string{"missing-command"}
		_, err := hc.formatCommandHelp(commands)
		require.ErrorContains(t, err, "missing-command")
	}
	cmd := Command{}
	require.Equal(t, "SYNOPSIS\n\n    \n", cmd.formatIndependHelp())
	require.Empty(t, cmd.formatOptionsHelp([]string{"missing-option"}))
	require.Equal(t, FormatTAB, cmd.formatOption(Option{}))
}

func TestFilterHelpersPreserveOrderAndMatchBasenames(t *testing.T) {
	values := []string{"dir/a.txt", "dir/b.log", "c.txt"}
	want := []string{"dir/a.txt", "c.txt"}
	require.Equal(t, want, filter(values, func(s string) bool { return strings.HasSuffix(s, ".txt") }))
	require.Equal(t, want, filter2(values, ".txt", strings.HasSuffix))
	require.True(t, filterStr("abc", "a", strings.HasPrefix))
	require.Equal(t, want, filterStrsWithPattern(values, "*.txt"))
	require.Equal(t, want, filterStrsWithInclude(values, "*.txt"))
	require.Equal(t, []string{"dir/b.log"}, filterStrsWithExclude(values, "*.txt"))
	require.True(t, containsInStrsSlice(values, "c.txt"))
	require.False(t, containsInStrsSlice(values, "other"))
	require.False(t, containsInStrsSlice(nil, "other"))
	filters := []filterOptionType{{IncludePrompt, "*.txt"}, {ExcludePrompt, "a*"}, {IncludePrompt, "*.log"}}
	expected := []string{"c.txt", "dir/b.log"}
	require.Equal(t, expected, matchFiltersForStrsInArray(values, filters))
	require.Equal(t, values, matchFiltersForStrsInArray(values, nil))
	require.Equal(t, []string{"dir/b.log"}, matchFiltersForStrsInArray(values, []filterOptionType{{ExcludePrompt, "*.txt"}}))
	require.True(t, matchFiltersForStr("file.txt", []filterOptionType{{ExcludePrompt, "*.log"}, {IncludePrompt, "*.txt"}}))
	require.False(t, matchFiltersForStr("file.txt", []filterOptionType{{ExcludePrompt, "*.txt"}}))
	require.Equal(t, want, filterObjectsFromListResultWithPattern(oss.ListObjectsResult{Objects: []oss.ObjectProperties{{Key: values[0]}, {Key: values[1]}, {Key: values[2]}}}, "*.txt"))
	source, destination := make(chan string, len(values)), make(chan string, len(values))
	for _, value := range values {
		source <- value
	}
	close(source)
	filterObjectsFromChanWithPattern(source, "*.txt", destination)
	var got []string
	for value := range destination {
		got = append(got, value)
	}
	require.Equal(t, want, got)
	files := []fileInfoType{{filePath: values[0]}, {filePath: values[1]}, {filePath: values[2]}}
	require.Equal(t, []fileInfoType{files[0], files[2]}, filterFilesWithInclude(files, "*.txt"))
	require.Equal(t, []fileInfoType{files[1]}, filterFilesWithExclude(files, "*.txt"))
	require.True(t, containsInFileSlice(files, files[0]))
	require.False(t, containsInFileSlice(files, fileInfoType{filePath: "missing"}))
	require.False(t, containsInFileSlice(nil, files[0]))
	objects := []objectInfoType{{relativeKey: values[0]}, {relativeKey: values[1]}, {relativeKey: values[2]}}
	require.Equal(t, []objectInfoType{objects[0], objects[2]}, filterObjectsWithInclude(objects, "*.txt"))
	require.Equal(t, []objectInfoType{objects[1]}, filterObjectsWithExclude(objects, "*.txt"))
	require.Equal(t, objects, matchFiltersForObjects(objects, nil))
	require.Equal(t, []objectInfoType{objects[2], objects[1]}, matchFiltersForObjects(objects, filters))
	require.Equal(t, []objectInfoType{objects[1]}, matchFiltersForObjects(objects, []filterOptionType{{ExcludePrompt, "*.txt"}}))
	srcObjects, dstObjects := make(chan objectInfoType, 3), make(chan objectInfoType, 3)
	makeObjectChanFromArray(objects, srcObjects)
	close(srcObjects)
	filterObjectsFromChanWithPatterns(srcObjects, filters, dstObjects)
	require.Equal(t, []objectInfoType{objects[2], objects[1]}, getObjectsFromChanToArray(dstObjects))
	valid, _ := getFilter([]string{"--include-prefix=ignored", "--include"})
	require.True(t, valid)
}

func TestLocalFileEnumeration(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "child"), 0700))
	for _, name := range []string{"a.txt", "b.log", "child/c.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0600))
	}
	for _, current := range []bool{true, false} {
		ch := make(chan fileInfoType, 20)
		require.NoError(t, getFileListCommon(dir, ch, current, false, false, []filterOptionType{{IncludePrompt, "*.txt"}}))
		var names []string
		for file := range ch {
			names = append(names, filepath.ToSlash(file.filePath))
		}
		if current {
			require.Equal(t, []string{"a.txt"}, names)
		} else {
			require.ElementsMatch(t, []string{"a.txt", "child/", "child/c.txt"}, names)
		}
	}
	for _, current := range []bool{true, false} {
		require.Error(t, getFileListCommon(filepath.Join(dir, "missing"), make(chan fileInfoType, 1), current, false, false, nil))
	}
	if err := os.Symlink(filepath.Join(dir, "child"), filepath.Join(dir, "link")); err == nil {
		for _, current := range []bool{true, false} {
			ch := make(chan fileInfoType, 20)
			require.NoError(t, getFileListCommon(dir, ch, current, true, false, nil))
			for file := range ch {
				require.NotEqual(t, "link", file.filePath)
			}
		}
		ch := make(chan fileInfoType, 20)
		require.NoError(t, getFileListCommon(dir, ch, false, false, true, nil))
		var names []string
		for file := range ch {
			names = append(names, filepath.ToSlash(file.filePath))
		}
		require.Contains(t, names, "link/c.txt")
	}
}

func TestStorageURLAndSmallUtilityBoundaries(t *testing.T) {
	for _, tc := range []struct{ input, encoding, want string }{
		{"oss://bucket/a%20b", URLEncodingType, "oss://bucket/a b"},
		{"a%20b", URLEncodingType, "a b"},
		{"~/file", "", filepath.Join(currentHomeDir(), "file")},
	} {
		u, err := StorageURLFromString(tc.input, tc.encoding)
		require.NoError(t, err)
		require.Equal(t, tc.want, u.ToString())
	}
	for _, input := range []string{"oss://bucket/%zz", "oss://%62ucket", "%zz"} {
		_, err := StorageURLFromString(input, URLEncodingType)
		require.Error(t, err)
	}
	for _, object := range []string{"/key", "\\key"} {
		u := CloudURL{object: object}
		require.Error(t, u.checkObjectPrefix())
	}
	var cu CloudURL
	require.NoError(t, cu.Init("/bucket/key", ""))
	require.Equal(t, "oss://bucket/key", cu.ToString())
	for _, input := range []string{"local-file", "oss://", "oss://bucket/%zz"} {
		_, err := GetCloudUrl(input, URLEncodingType)
		require.Error(t, err)
	}
	cloud, err := GetCloudUrl("oss://bucket/key", "")
	require.NoError(t, err)
	require.Equal(t, "key", cloud.object)
	require.Equal(t, int64(2), max(1, 2))
	require.Equal(t, int64(2), max(2, 1))
	require.Equal(t, "-1,234,567", getSizeString(-1234567))
	require.Equal(t, "0", getSizeString(0))
	require.Equal(t, -1, FindPos("missing", []string{"value"}))
	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.True(t, utcToLocalTime(when).Equal(when))
	h := fnv.New64()
	h.Write([]byte("key"))
	partition := int(h.Sum64() % 3)
	require.True(t, matchHash(h, "key", partition, 3))
	require.False(t, matchHash(h, "key", (partition+1)%3, 3))
	path, temporary := getBinaryPath()
	require.Equal(t, ".temp_"+filepath.Base(path), temporary)
}

func TestCommandConfigurationLookupAndClientValidation(t *testing.T) {
	cmd := Command{options: OptionMapType{}, configOptions: OptionMapType{}}
	for _, get := range []func() (string, bool){cmd.getEcsRamAkService, cmd.getRamRoleArn, cmd.getExternalId} {
		value, ok := get()
		require.Empty(t, value)
		require.False(t, ok)
	}
	for _, value := range []string{"", "configured"} {
		cmd.configOptions = OptionMapType{AkServiceSection: map[string]string{ItemEcsAk: value}, CREDSection: map[string]string{ItemRamRoleArn: value, ItemExternalId: value}}
		for _, get := range []func() (string, bool){cmd.getEcsRamAkService, cmd.getRamRoleArn, cmd.getExternalId} {
			got, ok := get()
			require.Equal(t, value, got)
			require.Equal(t, value != "", ok)
		}
	}
	cmd.configOptions = OptionMapType{BucketCnameSection: map[string]string{"bucket": "cname"}, BucketEndpointSection: map[string]string{"other": "endpoint"}}
	endpoint, cname := cmd.getEndpoint("bucket")
	require.Equal(t, "cname", endpoint)
	require.True(t, cname)
	endpoint, cname = cmd.getEndpoint("other")
	require.Equal(t, "endpoint", endpoint)
	require.False(t, cname)
	for _, missing := range []int{0, 1, 2} {
		values := []string{"endpoint", "id", "secret"}
		values[missing] = " "
		require.Error(t, cmd.checkCredentials(values[0], values[1], values[2]))
	}
	require.NoError(t, cmd.checkCredentials("endpoint", "id", "secret"))
	for _, tc := range []struct{ name, value, want string }{
		{OptionMode, "StsToken", "stsToken is empty"},
		{OptionMode, "RamRoleArn", "ramRoleArn is empty"},
		{OptionMode, "EcsRamRole", "ecsUrl is empty"},
		{OptionConnectTimeout, "invalid", "invalid syntax"},
		{OptionReadTimeout, "invalid", "invalid syntax"},
		{OptionSignVersion, "v4", "please enter the region"},
		{OptionMaxUpSpeed, "-1", "less than 0"},
		{OptionMaxDownSpeed, "-1", "less than 0"},
		{OptionLocalHost, "invalid host!", "net.ResolveIPAddr"},
	} {
		t.Run(tc.name+tc.value, func(t *testing.T) {
			_, opts, err := parseOSSOptions([]string{"--endpoint=http://127.0.0.1", "--access-key-id=id", "--access-key-secret=secret"})
			require.NoError(t, err)
			opts[tc.name] = &tc.value
			cmd := Command{options: opts, configOptions: OptionMapType{}}
			_, err = cmd.ossClient("bucket")
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestSTSSignatureAndLocalResponses(t *testing.T) {
	old := StsHost
	t.Cleanup(func() { StsHost = old })
	client := NewClient("id", "secret", "role", "session", "external")
	signed, err := client.generateSignedURL(900)
	require.NoError(t, err)
	u, err := url.Parse(signed)
	require.NoError(t, err)
	query := u.Query()
	require.Equal(t, "external", query.Get("ExternalId"))
	require.Equal(t, "900", query.Get("DurationSeconds"))
	signature := query.Get("Signature")
	query.Del("Signature")
	hash := hmac.New(sha1.New, []byte("secret&"))
	hash.Write([]byte("GET&%2F&" + url.QueryEscape(query.Encode())))
	require.Equal(t, base64.StdEncoding.EncodeToString(hash.Sum(nil)), signature)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "AssumeRole", r.URL.Query().Get("Action"))
		io.WriteString(w, `{"Credentials":{"AccessKeyId":"temporary","AccessKeySecret":"temporary-secret","SecurityToken":"token","Expiration":"2026-10-15T12:00:00Z"},"RequestId":"request"}`)
	}))
	defer server.Close()
	resp, err := client.AssumeRole(900, server.URL)
	require.NoError(t, err)
	require.Equal(t, "temporary", resp.Credentials.AccessKeyId)
	require.Equal(t, "token", resp.Credentials.SecurityToken)
	_, err = client.handleResponse([]byte(`{"Code":"Denied","Message":"no access","RequestId":"request"}`), 403)
	var serviceErr *ServiceError
	require.ErrorAs(t, err, &serviceErr)
	require.Equal(t, 403, serviceErr.StatusCode)
	require.Contains(t, err.Error(), "Denied")
	for _, status := range []int{200, 500} {
		_, err = client.handleResponse([]byte("invalid"), status)
		require.Error(t, err)
	}
	_, _, err = client.sendRequest("://invalid")
	require.Error(t, err)
	client.SessionName = "%zz"
	_, err = client.generateSignedURL(900)
	require.Error(t, err)
}

func TestOSSClientAuthenticationModesAndTransportOptions(t *testing.T) {
	oldHost := StsHost
	t.Cleanup(func() { StsHost = oldHost })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "role", r.URL.Query().Get("RoleArn"))
		io.WriteString(w, `{"Credentials":{"AccessKeyId":"sts-id","AccessKeySecret":"sts-secret","SecurityToken":"sts-token","Expiration":"2026-10-15T12:00:00Z"}}`)
	}))
	defer server.Close()
	StsHost = server.URL
	for _, mode := range []string{"AK", "StsToken", "RamRoleArn", "EcsRamRole", ""} {
		cmd := localOSSCommand(t, server.URL)
		cmd.options[OptionMode] = &mode
		cmd.options[OptionSTSToken] = stringPointer("token")
		cmd.options[OptionRamRoleArn] = stringPointer("role")
		cmd.options[OptionExternalId] = stringPointer("external")
		cmd.options[OptionECSRoleName] = stringPointer("ecs-role")
		client, err := cmd.ossClient("bucket")
		require.NoError(t, err)
		if mode == "StsToken" || mode == "" {
			require.Equal(t, "token", client.Config.SecurityToken)
		}
		if mode == "RamRoleArn" {
			require.Equal(t, "sts-id", client.Config.AccessKeyID)
			require.Equal(t, "sts-token", client.Config.SecurityToken)
		}
		if mode == "EcsRamRole" {
			require.Empty(t, client.Config.AccessKeyID)
			require.NotNil(t, client.Config.CredentialsProvider)
		}
	}
	cmd := localOSSCommand(t, server.URL)
	cmd.options[OptionMode] = stringPointer("RamRoleArn")
	cmd.configOptions[CREDSection] = map[string]string{ItemRamRoleArn: "role", ItemExternalId: "external"}
	cmd.options[OptionRoleSessionName] = stringPointer("session")
	client, err := cmd.ossClient("bucket")
	require.NoError(t, err)
	require.Equal(t, "sts-id", client.Config.AccessKeyID)
	cmd.options[OptionTokenTimeout] = stringPointer("invalid")
	_, err = cmd.ossClient("bucket")
	require.Error(t, err)
	cmd = localOSSCommand(t, server.URL)
	cmd.options[OptionAccessKeyID], cmd.options[OptionAccessKeySecret] = stringPointer(""), stringPointer("")
	cmd.configOptions[AkServiceSection] = map[string]string{ItemEcsAk: server.URL}
	client, err = cmd.ossClient("bucket")
	require.NoError(t, err)
	require.Empty(t, client.Config.AccessKeyID)
	for _, proxyUser := range []string{"", "user"} {
		cmd = localOSSCommand(t, server.URL)
		cmd.options[OptionRegion] = stringPointer("cn-test")
		cmd.options[OptionSignVersion] = stringPointer("v4")
		cmd.options[OptionCloudBoxID] = stringPointer("cloudbox")
		cmd.options[OptionProxyHost] = stringPointer("http://127.0.0.1:12345")
		cmd.options[OptionProxyUser] = &proxyUser
		cmd.options[OptionProxyPwd] = stringPointer("proxy-secret")
		cmd.options[OptionLocalHost] = stringPointer("127.0.0.1")
		cmd.options[OptionSkipVerifyCert] = boolPointer(true)
		cmd.options[OptionPassword] = boolPointer(true)
		cmd.options[OptionConnectTimeout] = stringPointer("3")
		cmd.options[OptionReadTimeout] = stringPointer("4")
		cmd.options[OptionMaxUpSpeed], cmd.options[OptionMaxDownSpeed] = stringPointer("1280"), stringPointer("1280")
		cmd.inputKeySecret = "input-secret"
		client, err = cmd.ossClient("bucket")
		require.NoError(t, err)
		require.Equal(t, "cn-test", client.Config.Region)
		require.Equal(t, "cloudbox", client.Config.CloudBoxId)
		require.Equal(t, oss.AuthV4, client.Config.AuthVersion)
		require.Equal(t, "input-secret", client.Config.AccessKeySecret)
		require.Equal(t, proxyUser != "", client.Config.IsAuthProxy)
		require.True(t, client.Config.IsUseProxy)
		require.True(t, client.Config.InsecureSkipVerify)
		require.Equal(t, 1280, client.Config.UploadLimitSpeed)
		require.Equal(t, 1280, client.Config.DownloadLimitSpeed)
		require.NotNil(t, client.Config.LocalAddr)
	}
}

func TestOSSCommandConfigurationAndArgumentValidation(t *testing.T) {
	cmd := Command{minArgc: 2, maxArgc: 3}
	require.ErrorContains(t, cmd.checkArgs(), "at least 2 arguments")
	cmd.args = []string{"1", "2", "3", "4"}
	require.ErrorContains(t, cmd.checkArgs(), "at most 3 arguments")
	cmd.options = OptionMapType{}
	require.True(t, cmd.needConfigFile())
	for _, name := range []string{OptionEndpoint, OptionAccessKeyID, OptionAccessKeySecret, OptionSTSToken} {
		cmd.options = OptionMapType{name: stringPointer("value")}
		require.False(t, cmd.needConfigFile())
	}
	queries := []string{"first:1"}
	for name, value := range (OptionMapType{OptionForce: boolPointer(true), OptionQueryParam: &queries, OptionEndpoint: stringPointer("endpoint")}) {
		cmd.options = OptionMapType{name: value}
		require.ErrorContains(t, cmd.checkOptions(), "does not support option")
		cmd.validOptionNames = []string{name}
		require.NoError(t, cmd.checkOptions())
		cmd.validOptionNames = nil
	}
	cmd.options = OptionMapType{OptionEndpoint: stringPointer("command-endpoint")}
	cmd.configOptions = OptionMapType{OptionEndpoint: "config-endpoint", BucketCnameSection: map[string]string{"bucket": "cname"}, BucketEndpointSection: map[string]string{"bucket": "endpoint"}}
	cmd.assembleOptions(nil)
	require.NotContains(t, cmd.configOptions, BucketCnameSection)
	require.NotContains(t, cmd.configOptions, BucketEndpointSection)
}
