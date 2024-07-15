package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// group spec text of all commands
const (
	GroupTypeNormalCommand string = "\nCommands:\n"

	GroupTypeAdditionalCommand string = "\nAdditional Commands:\n"

	GroupTypeDeprecatedCommand string = "\nDeprecated Commands:\n"
)

// CommandGroups is the array of all group types
var CommandGroups = []string{
	GroupTypeNormalCommand,
	GroupTypeAdditionalCommand,
	GroupTypeDeprecatedCommand,
}

// SpecText is the spec text of a command
type SpecText struct {
	synopsisText   string
	paramText      string
	syntaxText     string
	detailHelpText string
	sampleText     string
}

// Command contains all elements of a command, it's the base class of all commands
type Command struct {
	name             string
	nameAlias        []string
	minArgc          int
	maxArgc          int
	specChinese      SpecText
	specEnglish      SpecText
	validOptionNames []string
	group            string
	args             []string
	options          OptionMapType
	configOptions    OptionMapType
	inputKeySecret   string
}

// Commander is the interface of all commands
type Commander interface {
	RunCommand() error
	Init(args []string, options OptionMapType) error
}

// RewriteLoadConfiger is the interface for those commands, which do not need to load config, or have other action
type RewriteLoadConfiger interface {
	rewriteLoadConfig(string) error
}

// RewriteAssembleOptioner is the interface for those commands, which do not need to assemble options
type RewriteAssembleOptioner interface {
	rewriteAssembleOptions()
}

// Init is the common functions for all commands, they use Init to initialize itself
func (cmd *Command) Init(args []string, options OptionMapType, cmder interface{}) error {
	cmd.args = args
	cmd.options = options
	cmd.configOptions = OptionMapType{}

	if err := cmd.checkArgs(); err != nil {
		return err
	}

	val, _ := GetString(OptionConfigFile, cmd.options)
	if err := cmd.loadConfig(val, cmder); err != nil {
		return err
	}

	if err := cmd.checkOptions(); err != nil {
		return err
	}

	cmd.assembleOptions(cmder)
	return nil
}

func (cmd *Command) checkArgs() error {
	str := ""
	if cmd.minArgc > 1 {
		str = "s"
	}
	if len(cmd.args) < cmd.minArgc {
		msg := fmt.Sprintf("the command needs at least %d argument%s", cmd.minArgc, str)
		return CommandError{cmd.name, msg}
	}

	str = ""
	if cmd.minArgc > 1 {
		str = "s"
	}

	if len(cmd.args) > cmd.maxArgc {
		msg := fmt.Sprintf("the command needs at most %d argument%s", cmd.maxArgc, str)
		return CommandError{cmd.name, msg}
	}
	return nil
}

func (cmd *Command) loadConfig(configFile string, cmder interface{}) error {
	if cmdder, ok := cmder.(RewriteLoadConfiger); ok {
		return cmdder.rewriteLoadConfig(configFile)
	}
	var err error
	if cmd.configOptions, err = LoadConfig(configFile); err != nil && cmd.needConfigFile() {
		return err
	}
	return nil
}

func (cmd *Command) needConfigFile() bool {
	for _, name := range []string{OptionEndpoint, OptionAccessKeyID, OptionAccessKeySecret, OptionSTSToken} {
		val, _ := GetString(name, cmd.options)
		if val != "" {
			return false
		}
	}
	return true
}

func (cmd *Command) checkOptions() error {
	for name := range cmd.options {
		msg := fmt.Sprintf("the command does not support option: \"%s\"", name)
		switch OptionMap[name].optionType {
		case OptionTypeFlagTrue:
			if val, _ := GetBool(name, cmd.options); val {
				if FindPos(name, cmd.validOptionNames) == -1 {
					return CommandError{cmd.name, msg}
				}
			}
		case OptionTypeStrings:
			if val, _ := GetStrings(name, cmd.options); len(val) > 0 {
				if FindPos(name, cmd.validOptionNames) == -1 {
					return CommandError{cmd.name, msg}
				}
			}
		default:
			if val, _ := GetString(name, cmd.options); val != "" {
				if FindPos(name, cmd.validOptionNames) == -1 {
					return CommandError{cmd.name, msg}
				}
			}
		}
	}
	return nil
}

func (cmd *Command) assembleOptions(cmder interface{}) {
	if cmdder, ok := cmder.(RewriteAssembleOptioner); ok {
		cmdder.rewriteAssembleOptions()
		return
	}

	for name, option := range cmd.configOptions {
		if _, ok := cmd.options[name]; ok {
			if OptionMap[name].optionType != OptionTypeFlagTrue {
				if val, _ := GetString(name, cmd.options); val == "" {
					opval := option.(string)
					cmd.options[name] = &opval
					delete(cmd.configOptions, name)
				} else if name == OptionEndpoint {
					delete(cmd.configOptions, BucketCnameSection)
					delete(cmd.configOptions, BucketEndpointSection)
				}
			}
		}
	}

	for name := range cmd.options {
		if OptionMap[name].def != "" {
			switch OptionMap[name].optionType {
			case OptionTypeInt64:
				if val, _ := GetString(name, cmd.options); val == "" {
					def, _ := strconv.ParseInt(OptionMap[name].def, 10, 64)
					cmd.options[name] = &def
				}
			case OptionTypeAlternative:
				fallthrough
			case OptionTypeString:
				if val, _ := GetString(name, cmd.options); val == "" {
					def := OptionMap[name].def
					cmd.options[name] = &def
				}
			}
		}
	}
}

// FormatHelper is the interface for all commands to format spec information
type FormatHelper interface {
	formatHelpForWhole() string
	formatIndependHelp() string
}

func (cmd *Command) formatHelpForWhole() string {
	formatStr := "  %-" + strconv.Itoa(MaxCommandNameLen) + "s%s\n%s%s%s\n"
	spec := cmd.getSpecText()
	return fmt.Sprintf(formatStr, cmd.name, spec.paramText, FormatTAB, FormatTAB, spec.synopsisText)
}

func (cmd *Command) getSpecText() SpecText {
	val, _ := GetString(OptionLanguage, helpCommand.command.options)
	switch strings.ToLower(val) {
	case LEnglishLanguage:
		return cmd.specEnglish
	default:
		return cmd.specChinese
	}
}

func (cmd *Command) formatIndependHelp() string {
	spec := cmd.getSpecText()
	text := fmt.Sprintf("SYNOPSIS\n\n%s%s\n", FormatTAB, strings.TrimSpace(spec.synopsisText))
	if spec.syntaxText != "" {
		text += fmt.Sprintf("\nSYNTAX\n\n%s%s\n", FormatTAB, strings.TrimSpace(spec.syntaxText))
	}
	if spec.detailHelpText != "" {
		text += fmt.Sprintf("\nDETAIL DESCRIPTION\n\n%s%s\n", FormatTAB, strings.TrimSpace(spec.detailHelpText))
	}
	if spec.sampleText != "" {
		text += fmt.Sprintf("\nSAMPLE\n\n%s%s\n", FormatTAB, strings.TrimSpace(spec.sampleText))
	}
	if len(cmd.validOptionNames) != 0 {
		text += fmt.Sprintf("\nOPTIONS\n\n%s\n", cmd.formatOptionsHelp(cmd.validOptionNames))
	}
	return text
}

func (cmd *Command) formatOptionsHelp(validOptionNames []string) string {
	text := ""
	for _, optionName := range validOptionNames {
		if option, ok := OptionMap[optionName]; ok {
			text += cmd.formatOption(option)
		}
	}
	return text
}

func (cmd *Command) formatOption(option Option) string {
	text := FormatTAB
	if option.name != "" {
		text += option.name
		if option.def != "" {
			text += " " + option.def
		}
	}

	if option.name != "" && option.nameAlias != "" {
		text += ", "
	}

	if option.nameAlias != "" {
		text += option.nameAlias
		if option.def != "" {
			text += fmt.Sprintf("=%s", option.def)
		}
	}

	val, _ := GetString(OptionLanguage, helpCommand.command.options)
	val = strings.ToLower(val)
	opHelp := option.getHelp(val)
	if opHelp != "" {
		text += fmt.Sprintf("\n%s%s%s\n\n", FormatTAB, FormatTAB, opHelp)
	}

	return text
}

// OSS common function
// get oss client according to bucket(if bucket not empty)
func (cmd *Command) ossClient(bucket string) (*oss.Client, error) {
	endpoint, isCname := cmd.getEndpoint(bucket)
	accessKeyID, _ := GetString(OptionAccessKeyID, cmd.options)
	accessKeySecret, _ := GetString(OptionAccessKeySecret, cmd.options)
	stsToken, _ := GetString(OptionSTSToken, cmd.options)
	disableCRC64, _ := GetBool(OptionDisableCRC64, cmd.options)
	proxyHost, _ := GetString(OptionProxyHost, cmd.options)
	proxyUser, _ := GetString(OptionProxyUser, cmd.options)
	proxyPwd, _ := GetString(OptionProxyPwd, cmd.options)

	mode, _ := GetString(OptionMode, cmd.options)
	ecsRoleName, _ := GetString(OptionECSRoleName, cmd.options)

	strTokenTimeout, _ := GetString(OptionTokenTimeout, cmd.options)
	ramRoleArn, _ := GetString(OptionRamRoleArn, cmd.options)
	roleSessionName, _ := GetString(OptionRoleSessionName, cmd.options)

	strReadTimeout, _ := GetString(OptionReadTimeout, cmd.options)
	strConnectTimeout, _ := GetString(OptionConnectTimeout, cmd.options)

	stsRegion, _ := GetString(OptionSTSRegion, cmd.options)

	ecsUrl := ""

	localHost, _ := GetString(OptionLocalHost, cmd.options)
	bSkipVerifyCert, _ := GetBool(OptionSkipVerifyCert, cmd.options)
	region, _ := GetString(OptionRegion, cmd.options)
	signVersion, _ := GetString(OptionSignVersion, cmd.options)
	cloudBoxID, _ := GetString(OptionCloudBoxID, cmd.options)

	bPassword, _ := GetBool(OptionPassword, cmd.options)
	bForcePathStyle, _ := GetBool(OptionForcePathStyle, cmd.options)

	if bPassword {
		if cmd.inputKeySecret == "" {
			strPwd, err := GetPassword("input access key secret:")
			fmt.Printf("\r")
			if err != nil {
				return nil, err
			}
			cmd.inputKeySecret = string(strPwd)
		}
		accessKeySecret = cmd.inputKeySecret
	}

	options := []oss.ClientOption{}
	if region != "" {
		options = append(options, oss.Region(region))
	}

	if signVersion != "" {
		if strings.EqualFold(signVersion, "v4") {
			if region == "" {
				return nil, fmt.Errorf("In the v4 signature scenario, please enter the region")
			}
		}
		options = append(options, oss.AuthVersion(oss.AuthVersionType(signVersion)))
	}

	if cloudBoxID != "" {
		options = append(options, oss.CloudBoxId(cloudBoxID))
	}

	if strings.EqualFold(mode, "AK") {
		if err := cmd.checkCredentials(endpoint, accessKeyID, accessKeySecret); err != nil {
			return nil, err
		}
	} else if strings.EqualFold(mode, "StsToken") {

		if err := cmd.checkCredentials(endpoint, accessKeyID, accessKeySecret); err != nil {
			return nil, err
		}
		if stsToken == "" {
			return nil, fmt.Errorf("stsToken is empty")
		}
		options = append(options, oss.SecurityToken(stsToken))

	} else if strings.EqualFold(mode, "RamRoleArn") {
		if err := cmd.checkCredentials(endpoint, accessKeyID, accessKeySecret); err != nil {
			return nil, err
		}
		if ramRoleArn == "" {
			ramRoleArn, _ = cmd.getRamRoleArn()
		}
		if ramRoleArn == "" {
			return nil, fmt.Errorf("ramRoleArn is empty")
		}
		if roleSessionName == "" {
			roleSessionName = "SessNameRand" + randStr(5)
		}

		stsClient := NewClient(accessKeyID, accessKeySecret, ramRoleArn, roleSessionName)

		if strTokenTimeout == "" {
			strTokenTimeout = "3600"
		}
		intTokenTimeout, err := strconv.Atoi(strTokenTimeout)
		if err != nil {
			return nil, err
		}
		TokenTimeout := uint(intTokenTimeout)

		stsEndPoint := ""
		if stsRegion == "" {
			stsEndPoint = ""
		} else {
			stsEndPoint = "https://sts." + stsRegion + ".aliyuncs.com"
		}

		resp, err := stsClient.AssumeRole(TokenTimeout, stsEndPoint)
		if err != nil {
			return nil, err
		}

		accessKeyID = resp.Credentials.AccessKeyId
		accessKeySecret = resp.Credentials.AccessKeySecret
		stsToken = resp.Credentials.SecurityToken
		options = append(options, oss.SecurityToken(stsToken))
	} else if strings.EqualFold(mode, "EcsRamRole") {
		if ecsRoleName != "" {
			ecsUrl = "http://100.100.100.200/latest/meta-data/Ram/security-credentials/" + ecsRoleName
		} else {
			ecsUrl, _ = cmd.getEcsRamAkService()
		}

		if ecsUrl == "" {
			return nil, fmt.Errorf("ecsUrl is empty")
		}
		ecsRoleAKBuild := EcsRoleAKBuild{url: ecsUrl}
		options = append(options, oss.SetCredentialsProvider(&ecsRoleAKBuild))
		accessKeyID = ""
		accessKeySecret = ""

	} else if mode == "" {
		ecsUrl, _ = cmd.getEcsRamAkService()
		if accessKeyID == "" && ecsUrl == "" {
			return nil, fmt.Errorf("accessKeyID and ecsUrl are both empty")
		}
		if ecsUrl == "" {
			if err := cmd.checkCredentials(endpoint, accessKeyID, accessKeySecret); err != nil {
				return nil, err
			}
		}
		if accessKeyID == "" {
			LogInfo("using user ak service:%s\n", ecsUrl)
			ecsRoleAKBuild := EcsRoleAKBuild{url: ecsUrl}
			options = append(options, oss.SetCredentialsProvider(&ecsRoleAKBuild))
		}

		if stsToken != "" {
			options = append(options, oss.SecurityToken(stsToken))
		}
	}

	if strConnectTimeout == "" {
		strConnectTimeout = "120"
	}
	if strReadTimeout == "" {
		strReadTimeout = "1200"
	}
	connectTimeout, err := strconv.ParseInt(strConnectTimeout, 10, 64)
	if err != nil {
		return nil, err
	}
	readTimeout, err := strconv.ParseInt(strReadTimeout, 10, 64)
	if err != nil {
		return nil, err
	}

	userAgent, _ := GetString(OptionUserAgent, cmd.options)
	options = append(options, oss.UseCname(isCname), oss.UserAgent(getUserAgent(userAgent)), oss.Timeout(connectTimeout, readTimeout))

	if disableCRC64 {
		options = append(options, oss.EnableCRC(false))
	} else {
		options = append(options, oss.EnableCRC(true))
	}

	if proxyHost != "" {
		if proxyUser != "" {
			options = append(options, oss.AuthProxy(proxyHost, proxyUser, proxyPwd))
		} else {
			options = append(options, oss.Proxy(proxyHost))
		}
	}

	if localHost != "" {
		ipAddr, err := net.ResolveIPAddr("ip", localHost)
		if err != nil {
			return nil, fmt.Errorf("net.ResolveIPAddr error,%s", err.Error())
		}
		localTCPAddr := &(net.TCPAddr{IP: ipAddr.IP})
		options = append(options, oss.SetLocalAddr(localTCPAddr))
	}

	if logLevel > oss.LogOff {
		options = append(options, oss.SetLogLevel(logLevel))
		options = append(options, oss.SetLogger(utilLogger))
	}

	if bSkipVerifyCert {
		LogInfo("skip verify oss server's tls certificate\n")
		options = append(options, oss.InsecureSkipVerify(true))
	}

	if bForcePathStyle {
		LogInfo("use path-style access instead of virtual hosted-style access.\n")
		options = append(options, oss.ForcePathStyle(true))
	}

	client, err := oss.New(endpoint, accessKeyID, accessKeySecret, options...)
	if err != nil {
		return nil, err
	}

	maxUpSpeed, errUp := GetInt(OptionMaxUpSpeed, cmd.options)
	if errUp == nil {
		if maxUpSpeed >= 0 {
			errUp = client.LimitUploadSpeed(int(maxUpSpeed))
			if errUp != nil {
				return nil, errUp
			} else {
				LogInfo("set maxupspeed success,value is %d(KB/s)\n", maxUpSpeed)
			}
		} else {
			return nil, fmt.Errorf("invalid value,maxupspeed %d less than 0", maxUpSpeed)
		}
	}

	maxDownSpeed, errDown := GetInt(OptionMaxDownSpeed, cmd.options)
	if errDown == nil {
		if maxDownSpeed >= 0 {
			errDown = client.LimitDownloadSpeed(int(maxDownSpeed))
			if errDown != nil {
				return nil, errDown
			} else {
				LogInfo("set maxdownspeed success,value is %d(KB/s)\n", maxDownSpeed)
			}
		} else {
			return nil, fmt.Errorf("invalid value,maxdownspeed %d less than 0", maxDownSpeed)
		}
	}

	return client, nil
}

func (cmd *Command) checkCredentials(endpoint, accessKeyID, accessKeySecret string) error {
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("invalid endpoint, endpoint is empty, please check your config")
	}
	if strings.TrimSpace(accessKeyID) == "" {
		return fmt.Errorf("invalid accessKeyID, accessKeyID is empty, please check your config")
	}
	if strings.TrimSpace(accessKeySecret) == "" {
		return fmt.Errorf("invalid accessKeySecret, accessKeySecret is empty, please check your config")
	}
	return nil
}

func (cmd *Command) getEcsRamAkService() (string, bool) {
	if urlMap, ok := cmd.configOptions[AkServiceSection]; ok {
		if strUrl, ok := urlMap.(map[string]string)[ItemEcsAk]; ok {
			if strUrl != "" {
				return strUrl, true
			} else {
				return "", false
			}
		}
	}
	return "", false
}

func (cmd *Command) getRamRoleArn() (string, bool) {
	if arnMap, ok := cmd.configOptions[CREDSection]; ok {
		if strArn, ok := arnMap.(map[string]string)[ItemRamRoleArn]; ok {
			if strArn != "" {
				return strArn, true
			} else {
				return "", false
			}
		}
	}
	return "", false
}

func (cmd *Command) getEndpoint(bucket string) (string, bool) {
	if cnameMap, ok := cmd.configOptions[BucketCnameSection]; ok {
		if endpoint, ok := cnameMap.(map[string]string)[bucket]; ok {
			return endpoint, true
		}
	}

	if eMap, ok := cmd.configOptions[BucketEndpointSection]; ok {
		if endpoint, ok := eMap.(map[string]string)[bucket]; ok {
			return endpoint, false
		}
	}

	endpoint, _ := GetString(OptionEndpoint, cmd.options)
	return endpoint, false
}

// get oss operable bucket
func (cmd *Command) ossBucket(bucketName string) (*oss.Bucket, error) {
	client, err := cmd.ossClient(bucketName)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return bucket, nil
}

func (cmd *Command) ossListObjectsRetry(bucket *oss.Bucket, options ...oss.Option) (oss.ListObjectsResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, cmd.options)
	for i := 1; ; i++ {
		lor, err := bucket.ListObjects(options...)
		if err == nil {
			return lor, err
		}

		// http 4XX error no need to retry
		// only network error or internal error need to retry
		serviceError, noNeedRetry := err.(oss.ServiceError)
		if int64(i) >= retryTimes || (noNeedRetry && serviceError.StatusCode < 500) {
			return lor, ObjectError{err, bucket.BucketName, ""}
		}

		// wait 1 second
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (cmd *Command) ossListObjectVersionsRetry(bucket *oss.Bucket, options ...oss.Option) (oss.ListObjectVersionsResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, cmd.options)
	for i := 1; ; i++ {
		lor, err := bucket.ListObjectVersions(options...)
		if err == nil {
			return lor, err
		}
		if int64(i) >= retryTimes {
			return lor, BucketError{err, bucket.BucketName}
		}
	}
}

func (cmd *Command) ossListMultipartUploadsRetry(bucket *oss.Bucket, options ...oss.Option) (oss.ListMultipartUploadResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, cmd.options)
	for i := 1; ; i++ {
		lmr, err := bucket.ListMultipartUploads(options...)
		if err == nil {
			return lmr, err
		}

		// http 4XX error no need to retry
		// only network error or internal error need to retry
		serviceError, noNeedRetry := err.(oss.ServiceError)
		if int64(i) >= retryTimes || (noNeedRetry && serviceError.StatusCode < 500) {
			return lmr, ObjectError{err, bucket.BucketName, ""}
		}

		// wait 1 second
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (cmd *Command) ossGetObjectStatRetry(bucket *oss.Bucket, object string, options ...oss.Option) (http.Header, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, cmd.options)
	for i := 1; ; i++ {
		props, err := bucket.GetObjectDetailedMeta(object, options...)
		if err == nil {
			return props, err
		}

		// http 4XX error no need to retry
		// only network error or internal error need to retry
		serviceError, noNeedRetry := err.(oss.ServiceError)
		if int64(i) >= retryTimes || (noNeedRetry && serviceError.StatusCode < 500) {
			return props, ObjectError{err, bucket.BucketName, object}
		}

		// wait 1 second
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (cmd *Command) ossGetObjectMetaRetry(bucket *oss.Bucket, object string, options ...oss.Option) (http.Header, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, cmd.options)
	for i := 1; ; i++ {
		props, err := bucket.GetObjectMeta(object, options...)
		if err == nil {
			return props, err
		}

		// http 4XX error no need to retry
		// only network error or internal error need to retry
		serviceError, noNeedRetry := err.(oss.ServiceError)
		if int64(i) >= retryTimes || (noNeedRetry && serviceError.StatusCode < 500) {
			return props, ObjectError{err, bucket.BucketName, object}
		}

		// wait 1 second
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (cmd *Command) objectStatistic(bucket *oss.Bucket, cloudURL CloudURL, monitor Monitorer, filters []filterOptionType, options ...oss.Option) {
	if monitor == nil {
		return
	}

	pre := oss.Prefix(cloudURL.object)
	marker := oss.Marker("")
	for {
		listOptions := append(options, marker, pre)
		lor, err := cmd.ossListObjectsRetry(bucket, listOptions...)
		if err != nil {
			monitor.setScanError(err)
			return
		}

		for _, object := range lor.Objects {
			if doesSingleObjectMatchPatterns(object.Key, filters) {
				monitor.updateScanNum(1)
			}
		}

		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}

	monitor.setScanEnd()
}

func (cmd *Command) objectProducer(bucket *oss.Bucket, cloudURL CloudURL, chObjects chan<- string, chError chan<- error, filters []filterOptionType, options ...oss.Option) {
	defer close(chObjects)
	pre := oss.Prefix(cloudURL.object)
	marker := oss.Marker("")
	for {
		listOptions := append(options, marker, pre)
		lor, err := cmd.ossListObjectsRetry(bucket, listOptions...)
		if err != nil {
			chError <- err
			return
		}
		for _, object := range lor.Objects {
			if doesSingleObjectMatchPatterns(object.Key, filters) {
				chObjects <- object.Key
			}
		}

		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}
	chError <- nil
}

func (cmd *Command) getRawMarker(str string) (string, error) {
	encodingType, _ := GetString(OptionEncodingType, cmd.options)
	if encodingType == URLEncodingType {
		unencodedStr, err := url.QueryUnescape(str)
		if err != nil {
			return str, err
		}
		return unencodedStr, nil
	}
	return str, nil
}

func (cmd *Command) updateMonitor(err error, monitor *Monitor) {
	if monitor == nil {
		return
	}
	if err == nil {
		monitor.updateOKNum(1)
	} else {
		monitor.updateErrNum(1)
	}
	fmt.Printf(monitor.progressBar(false, normalExit))
}

func (cmd *Command) report(msg string, err error, option *batchOptionType) {
	if cmd.filterError(err, option) {
		option.reporter.ReportError(fmt.Sprintf("%s error, info: %s", msg, err.Error()))
		option.reporter.Prompt(err)
	}
}

func (cmd *Command) filterError(err error, option *batchOptionType) bool {
	if err == nil {
		return false
	}

	errorTypeName := reflect.TypeOf(err).String()
	if !strings.Contains(errorTypeName, "ObjectError") {
		return false
	}

	err = err.(ObjectError).err

	switch err.(type) {
	case oss.ServiceError:
		code := err.(oss.ServiceError).Code
		if code == "NoSuchBucket" || code == "InvalidAccessKeyId" || code == "SignatureDoesNotMatch" || code == "AccessDenied" || code == "RequestTimeTooSkewed" || code == "InvalidBucketName" {
			option.ctnu = false
			return false
		}
	}
	return true
}

func (cmd *Command) getOSSOptions(hopMap map[string]interface{}, headers map[string]string) ([]oss.Option, error) {
	options := []oss.Option{}
	for name, value := range headers {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(oss.HTTPHeaderOssMetaPrefix)) {
			options = append(options, oss.Meta(name[len(oss.HTTPHeaderOssMetaPrefix):], value))
		} else {
			option, err := getOSSOption(hopMap, name, value)
			if err == nil {
				options = append(options, option)
			} else if strings.HasPrefix(strings.ToLower(name), "x-oss-") {
				options = append(options, oss.SetHeader(name, value))
			} else {
				return nil, err
			}
		}
	}
	return options, nil
}

func (cmd *Command) getOSSTagging(strTagging string) ([]oss.Tag, error) {
	tags := []oss.Tag{}
	strKeys := strings.Split(strTagging, "&")
	for _, v := range strKeys {
		if v == "" {
			return tags, fmt.Errorf("tagging value is empty,maybe exist &&")
		}
		tagNode := strings.Split(v, "=")
		if len(tagNode) >= 3 {
			return tags, fmt.Errorf("tagging value error %s", v)
		}

		// value maybe empty
		tagNode = append(tagNode, "")
		tags = append(tags, oss.Tag{
			Key:   tagNode[0],
			Value: tagNode[1],
		})
	}
	return tags, nil
}

// GetAllCommands returns all commands list
func GetAllCommands() []interface{} {
	return []interface{}{
		&helpCommand,
		&configCommand,
		&makeBucketCommand,
		&listCommand,
		&removeCommand,
		&statCommand,
		&setACLCommand,
		&setMetaCommand,
		&copyCommand,
		&restoreCommand,
		&createSymlinkCommand,
		&readSymlinkCommand,
		&signURLCommand,
		&hashCommand,
		&updateCommand,
		&probeCommand,
		&mkdirCommand,
		&corsCommand,
		&bucketLogCommand,
		&bucketRefererCommand,
		&listPartCommand,
		&allPartSizeCommand,
		&appendFileCommand,
		&catCommand,
		&bucketTagCommand,
		&bucketEncryptionCommand,
		&corsOptionsCommand,
		&bucketStyleCommand,
		&bucketLifeCycleCommand,
		&bucketWebsiteCommand,
		&bucketQosCommand,
		&userQosCommand,
		&bucketVersioningCommand,
		&duSizeCommand,
		&bucketPolicyCommand,
		&requestPaymentCommand,
		&objectTagCommand,
		&bucketInventoryCommand,
		&revertCommand,
		&syncCommand,
		&wormCommand,
		&lrbCommand,
		&replicationCommand,
		&bucketCnameCommand,
		&lcbCommand,
		&bucketAccessMonitorCommand,
		&bucketResourceGroupCommand,
	}
}
