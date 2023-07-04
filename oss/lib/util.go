package lib

import (
	"bytes"
	"fmt"
	"hash"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"golang.org/x/crypto/ssh/terminal"
)

var sys_name string
var sys_release string
var sys_machine string

func init() {
	sys_name = runtime.GOOS
	sys_release = "-"
	sys_machine = runtime.GOARCH

	if !strings.EqualFold(sys_name, "windows") {
		if out, err := exec.Command("uname", "-s").CombinedOutput(); err == nil {
			sys_name = string(bytes.TrimSpace(out))
		}
		if out, err := exec.Command("uname", "-r").CombinedOutput(); err == nil {
			sys_release = string(bytes.TrimSpace(out))
		}
		if out, err := exec.Command("uname", "-m").CombinedOutput(); err == nil {
			sys_machine = string(bytes.TrimSpace(out))
		}
	}
}

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
	return sysInfo{name: sys_name, release: sys_release, machine: sys_machine}
}

func getUserAgent(ua string) string {
	sys := getSysInfo()
	if ua == "" {
		return fmt.Sprintf("aliyun-sdk-go/%s (%s/%s/%s;%s)/%s-%s", oss.Version, sys.name, sys.release, sys.machine, runtime.Version(), Package, Version)
	}
	return fmt.Sprintf("aliyun-sdk-go/%s (%s/%s/%s;%s)/%s-%s/%s", oss.Version, sys.name, sys.release, sys.machine, runtime.Version(), Package, Version, ua)
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

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

func randStr(n int) string {
	b := make([]rune, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func currentHomeDir() string {
	homeDir := ""
	homeDrive := os.Getenv("HOMEDRIVE")
	homePath := os.Getenv("HOMEPATH")
	if runtime.GOOS == "windows" && homeDrive != "" && homePath != "" {
		homeDir = homeDrive + string(os.PathSeparator) + homePath
	}

	if homeDir != "" {
		return homeDir
	}

	usr, _ := user.Current()
	if usr != nil {
		homeDir = usr.HomeDir
	} else {
		homeDir = os.Getenv("HOME")
	}
	return homeDir
}

func getCurrentDirFileListCommon(dpath string, chFiles chan<- fileInfoType, filters []filterOptionType) error {
	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	fileList, err := ioutil.ReadDir(dpath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileList {
		if !fileInfo.IsDir() {
			realInfo, errF := os.Stat(dpath + fileInfo.Name())
			if errF == nil && realInfo.IsDir() {
				// for symlink
				continue
			}

			if doesSingleFileMatchPatterns(fileInfo.Name(), filters) {
				chFiles <- fileInfoType{fileInfo.Name(), dpath}
			}
		}
	}
	return nil
}

func getFileListCommon(dpath string, chFiles chan<- fileInfoType, onlyCurrentDir bool, disableAllSymlink bool,
	enableSymlinkDir bool, filters []filterOptionType) error {
	defer close(chFiles)
	if onlyCurrentDir {
		return getCurrentDirFileListCommon(dpath, chFiles, filters)
	}

	name := dpath
	symlinkDiretorys := []string{dpath}
	walkFunc := func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)

		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				if strings.HasSuffix(fileName, "\\") || strings.HasSuffix(fileName, "/") {
					chFiles <- fileInfoType{fileName, name}
				} else {
					chFiles <- fileInfoType{fileName + string(os.PathSeparator), name}
				}
			}
			return nil
		}

		if disableAllSymlink && (f.Mode()&os.ModeSymlink) != 0 {
			return nil
		}

		if enableSymlinkDir && (f.Mode()&os.ModeSymlink) != 0 {
			// there is difference between os.Stat and os.Lstat in filepath.Walk
			realInfo, err := os.Stat(fpath)
			if err != nil {
				return err
			}

			if realInfo.IsDir() {
				// it's symlink dir
				// if linkDir has suffix os.PathSeparator,os.Lstat determine it is a dir
				if !strings.HasSuffix(name, string(os.PathSeparator)) {
					name += string(os.PathSeparator)
				}
				linkDir := name + fileName + string(os.PathSeparator)
				symlinkDiretorys = append(symlinkDiretorys, linkDir)
				return nil
			}
		}

		if doesSingleFileMatchPatterns(fileName, filters) {
			chFiles <- fileInfoType{fileName, name}
		}
		return nil
	}

	var err error
	for {
		symlinks := symlinkDiretorys
		symlinkDiretorys = []string{}
		for _, v := range symlinks {
			err = filepath.Walk(v, walkFunc)
			if err != nil {
				return err
			}
		}
		if len(symlinkDiretorys) == 0 {
			break
		}
	}
	return err
}

func getObjectListCommon(bucket *oss.Bucket, cloudURL CloudURL, chObjects chan<- objectInfoType,
	onlyCurrentDir bool, filters []filterOptionType, payerOptions []oss.Option) error {
	defer close(chObjects)
	pre := oss.Prefix(cloudURL.object)
	marker := oss.Marker("")
	//while the src object is end with "/", use object key as marker, exclude the object itself
	if strings.HasSuffix(cloudURL.object, "/") {
		marker = oss.Marker(cloudURL.object)
	}
	del := oss.Delimiter("")
	if onlyCurrentDir {
		del = oss.Delimiter("/")
	}

	listOptions := append(payerOptions, pre, marker, del, oss.MaxKeys(1000))
	for {
		lor, err := bucket.ListObjects(listOptions...)
		if err != nil {
			return err
		}

		for _, object := range lor.Objects {
			prefix := ""
			relativeKey := object.Key
			index := strings.LastIndex(cloudURL.object, "/")
			if index > 0 {
				prefix = object.Key[:index+1]
				relativeKey = object.Key[index+1:]
			}

			if doesSingleObjectMatchPatterns(object.Key, filters) {
				if strings.ToLower(object.Type) == "symlink" {
					props, err := bucket.GetObjectDetailedMeta(object.Key, payerOptions...)
					if err != nil {
						LogError("ossGetObjectStatRetry error info:%s\n", err.Error())
						return err
					}
					size, err := strconv.ParseInt(props.Get(oss.HTTPHeaderContentLength), 10, 64)
					if err != nil {
						LogError("strconv.ParseInt error info:%s\n", err.Error())
						return err

					}
					object.Size = size
				}
				chObjects <- objectInfoType{prefix, relativeKey, int64(object.Size), object.LastModified}
			}
		}

		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		listOptions = append(payerOptions, pre, marker, oss.MaxKeys(1000))
		if !lor.IsTruncated {
			break
		}
	}
	return nil
}

func GetPassword(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		state, err := terminal.MakeRaw(fd)
		if err != nil {
			f := "getpass: cannot disable terminal echoing — %v"
			return nil, fmt.Errorf(f, err)
		}
		defer terminal.Restore(fd, state)
		defer fmt.Println()
	}

	if prompt == "" {
		prompt = "enter password: "
	}
	fmt.Fprint(os.Stderr, prompt)
	return terminal.ReadPassword(fd)
}

// AddStringsToOption add strings option to oss option
func AddStringsToOption(params []string, options []oss.Option) ([]oss.Option, error) {
	if params == nil {
		return options, nil
	}
	paramsMap := map[string]string{}
	for _, s := range params {
		pair := strings.SplitN(s, ":", 2)
		name := pair[0]
		value := ""
		if len(pair) > 1 {
			value = pair[1]
		}
		paramsMap[name] = value
	}
	for key, value := range paramsMap {
		options = append(options, oss.AddParam(key, value))
	}
	return options, nil
}
