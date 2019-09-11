package lib

import (
	"bytes"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Output print input string to stdout and add '\n'
func Output(str string) {
	fmt.Println(str)
}

// FindPos find the elem position in a string array
func FindPos(elem string, elemArray []string) int {
	for p, v := range elemArray {
		if v == elem {
			return p
		}
	}
	return -1
}

// FindPosCaseInsen find the elem position in a string array, ignore case
func FindPosCaseInsen(elem string, elemArray []string) int {
	for p, v := range elemArray {
		if strings.EqualFold(v, elem) {
			return p
		}
	}
	return -1
}

func getBinaryPath() (string, string) {
	filePath, _ := exec.LookPath(os.Args[0])
	if path, err := os.Readlink(filePath); err == nil {
		filePath = path
	}

	fileName := filepath.Base(filePath)
	renameFilePath := ".temp_" + fileName
	return filePath, renameFilePath
}

type sysInfo struct {
	name    string // 操作系统名称windows/Linux
	release string // 操作系统版本 2.6.32-220.23.2.ali1089.el5.x86_64等
	machine string // 机器类型amd64/x86_64
}

// Get　system info
// 获取操作系统信息、机器类型
func getSysInfo() sysInfo {
	name := runtime.GOOS
	release := "-"
	machine := runtime.GOARCH
	if out, err := exec.Command("uname", "-s").CombinedOutput(); err == nil {
		name = string(bytes.TrimSpace(out))
	}
	if out, err := exec.Command("uname", "-r").CombinedOutput(); err == nil {
		release = string(bytes.TrimSpace(out))
	}
	if out, err := exec.Command("uname", "-m").CombinedOutput(); err == nil {
		machine = string(bytes.TrimSpace(out))
	}
	return sysInfo{name: name, release: release, machine: machine}
}

func getUserAgent() string {
	sys := getSysInfo()
	return fmt.Sprintf("aliyun-sdk-go/%s (%s/%s/%s;%s)/%s-%s", oss.Version, sys.name, sys.release, sys.machine, runtime.Version(), Package, Version)
}

func utcToLocalTime(utc time.Time) time.Time {
	return utc.In(time.Local)
}

func max(a, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}

func getSizeString(size int64) string {
	prefix := ""
	str := fmt.Sprintf("%d", size)
	if size < 0 {
		prefix = "-"
		str = str[1:]
	}
	len := len(str)
	strList := []string{}
	i := len % 3
	if i != 0 {
		strList = append(strList, str[0:i])
	}
	for ; i < len; i += 3 {
		strList = append(strList, str[i:i+3])
	}
	return fmt.Sprintf("%s%s", prefix, strings.Join(strList, ","))
}

// Returns a new slice containing all strings in the
// slice that satisfy the predicate `f`.
func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// predicate `f` has 2 parameters
func filter2(vs []string, s string, f func(_, _ string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v, s) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func filterStr(v string, s string, f func(string, string) bool) bool {
	return f(v, s)
}

func filterStrWithPattern(v, p string) bool {
	_, name := filepath.Split(v)
	res, _ := filepath.Match(p, name)
	return res
}

func filterStrsWithPattern(vs []string, p string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		_, name := filepath.Split(v)
		res, _ := filepath.Match(p, name)
		if res {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func filterObjectsFromListResultWithPattern(lor oss.ListObjectsResult, pattern string) []string {
	objs := make([]string, 0)
	for _, obj := range lor.Objects {
		objs = append(objs, obj.Key)
	}
	return filterStrsWithPattern(objs, pattern)
}

func filterObjectsFromChanWithPattern(srcCh <-chan string, pattern string, dstCh chan<- string) {
	for obj := range srcCh {
		if filterStrWithPattern(obj, pattern) {
			dstCh <- obj
		}
	}
	defer close(dstCh)
}

// Following for strings
func getFilter(cmdline []string) (bool, []filterOptionType) {
	filters := make([]filterOptionType, 0)
	for i, item := range cmdline {
		var strTag = ""
		if strings.Index(item, IncludePrompt) == 0 {
			strTag = IncludePrompt
		} else if strings.Index(item, ExcludePrompt) == 0 {
			strTag = ExcludePrompt
		}

		if strTag != "" {
			var filter filterOptionType
			var strArg string

			filter.name = strTag
			if item == strTag {
				strArg = cmdline[i+1]
			} else if item[len(strTag)] == '=' {
				strArg = item[len(strTag)+1:]
			}

			if strArg == "" {
				continue
			}

			// To support standard glob
			filter.pattern = strings.Replace(strArg, "[!", "[^", -1)
			dir, _ := filepath.Split(filter.pattern)
			if dir != "" {
				return false, filters
			}
			filters = append(filters, filter)
		}
	}

	return true, filters
}

func containsInStrsSlice(vs []string, t string) bool {
	if len(vs) == 0 {
		return false
	}

	for _, v := range vs {
		if v == t {
			return true
		}
	}
	return false
}

func filterSingleStr(v, p string, include bool) bool {
	_, name := filepath.Split(v)
	res, _ := filepath.Match(p, name)

	if include {
		return res
	} else {
		return !res
	}
}

func filterStrsWithInclude(vs []string, p string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		_, name := filepath.Split(v)
		res, _ := filepath.Match(p, name)
		if res {
			vsf = append(vsf, v)
		}

	}
	return vsf
}

func filterStrsWithExclude(vs []string, p string) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		_, name := filepath.Split(v)
		res, _ := filepath.Match(p, name)
		if !res {
			vsf = append(vsf, v)
		}

	}
	return vsf
}

func matchFiltersForStr(str string, filters []filterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	var res bool
	if filters[0].name == IncludePrompt {
		res = filterSingleStr(str, filters[0].pattern, true)
	} else {
		res = filterSingleStr(str, filters[0].pattern, false)
	}

	for _, filter := range filters[1:] {
		if filter.name == IncludePrompt {
			res = res || filterSingleStr(str, filter.pattern, true)
		} else {
			res = res && filterSingleStr(str, filter.pattern, false)
		}
	}

	return res
}

func matchFiltersForStrs(strs []string, filters []filterOptionType) []string {
	if len(filters) == 0 {
		return strs
	}

	vsf := make([]string, 0)

	for _, str := range strs {
		if matchFiltersForStr(str, filters) {
			vsf = append(vsf, str)
		}
	}

	return vsf
}

func matchFiltersForStrsInArray(strs []string, filters []filterOptionType) []string {
	if len(filters) == 0 {
		return strs
	}
	vsf := make([]string, 0)

	if filters[0].name == IncludePrompt {
		vsf = append(vsf, filterStrsWithInclude(strs, filters[0].pattern)...)
	} else {
		vsf = append(vsf, filterStrsWithExclude(strs, filters[0].pattern)...)
	}

	for _, filter := range filters[1:] {
		if filter.name == IncludePrompt {
			vsf = append(vsf, filterStrsWithInclude(strs, filter.pattern)...)
		} else {
			vsf = filterStrsWithExclude(vsf, filter.pattern)
		}
	}

	return vsf
}

// Following for files

func doesSingleFileMatchPatterns(filename string, filters []filterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	files := []string{filename}
	vsf := matchFiltersForStrs(files, filters)

	if len(vsf) > 0 {
		return true
	}
	return false
}

func containsInFileSlice(vs []fileInfoType, t fileInfoType) bool {
	if len(vs) == 0 {
		return false
	}

	for _, v := range vs {
		if v == t {
			return true
		}
	}
	return false
}

func filterFilesWithInclude(vs []fileInfoType, p string) []fileInfoType {
	vsf := make([]fileInfoType, 0)
	for _, v := range vs {
		_, filename := filepath.Split(v.filePath)
		res, _ := filepath.Match(p, filename)

		if res {
			vsf = append(vsf, v)
		}

	}
	return vsf
}

func filterFilesWithExclude(vs []fileInfoType, p string) []fileInfoType {
	vsf := make([]fileInfoType, 0)
	for _, v := range vs {
		_, filename := filepath.Split(v.filePath)
		res, _ := filepath.Match(p, filename)
		if !res {
			vsf = append(vsf, v)
		}

	}
	return vsf
}

// Following for objects

func doesSingleObjectMatchPatterns(object string, filters []filterOptionType) bool {
	if len(filters) == 0 {
		return true
	}

	return doesSingleFileMatchPatterns(object, filters)
}

func getObjectsFromChanToArray(chObjects <-chan objectInfoType) []objectInfoType {
	objects := make([]objectInfoType, 0)
	for f := range chObjects {
		objects = append(objects, f)
	}
	return objects
}

func filterObjectsWithInclude(vs []objectInfoType, p string) []objectInfoType {
	vsf := make([]objectInfoType, 0)
	for _, v := range vs {
		_, key := filepath.Split(v.relativeKey)
		//_, key := filepath.Split(v.key)
		res, _ := filepath.Match(p, key)
		if res {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func filterObjectsWithExclude(vs []objectInfoType, p string) []objectInfoType {
	vsf := make([]objectInfoType, 0)
	for _, v := range vs {
		_, key := filepath.Split(v.relativeKey)
		//_, key := filepath.Split(v.key)
		res, _ := filepath.Match(p, key)
		if !res {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func matchFiltersForObjects(objects []objectInfoType, filters []filterOptionType) []objectInfoType {
	if len(filters) == 0 {
		return objects
	}

	vsf := make([]objectInfoType, 0)

	for i, filter := range filters {
		if filter.name == IncludePrompt {
			vsf = append(vsf, filterObjectsWithInclude(objects, filter.pattern)...)
		} else {
			if i == 0 {
				vsf = append(vsf, filterObjectsWithExclude(objects, filter.pattern)...)
			} else {
				vsf = filterObjectsWithExclude(vsf, filter.pattern)
			}
		}
	}

	return vsf
}

func makeObjectChanFromArray(objects []objectInfoType, chObjs chan<- objectInfoType) {
	for _, f := range objects {
		chObjs <- f
	}
}

func filterObjectsFromChanWithPatterns(chObjects <-chan objectInfoType, filters []filterOptionType, dstObjs chan<- objectInfoType) {
	objects := getObjectsFromChanToArray(chObjects)
	vsf := matchFiltersForObjects(objects, filters)
	makeObjectChanFromArray(vsf, dstObjs)
	defer close(dstObjs)
}

func GetCloudUrl(strlUrl, encodingType string) (*CloudURL, error) {
	bucketUrL, err := StorageURLFromString(strlUrl, encodingType)
	if err != nil {
		return nil, err
	}

	if !bucketUrL.IsCloudURL() {
		return nil, fmt.Errorf("parameter is not a cloud url,url is %s", bucketUrL.ToString())
	}

	cloudUrl := bucketUrL.(CloudURL)
	if cloudUrl.bucket == "" {
		return nil, fmt.Errorf("bucket name is empty,url is %s", bucketUrL.ToString())
	}
	return &cloudUrl, nil
}

func matchHash(fnvIns hash.Hash64, key string, modeValue int, countValue int) bool {
	fnvIns.Reset()
	fnvIns.Write([]byte(key))
	if fnvIns.Sum64()%uint64(countValue) == uint64(modeValue) {
		return true
	}
	return false
}
