package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
)

var (
	vUpdateEndpoint = updateEndpoint
	vUpdateBucket   = updateBucket
	vVersion        = Version
)

var specChineseUpdate = SpecText{

	synopsisText: "更新ossutil",

	paramText: "[options]",

	syntaxText: ` 
    ossutil update [-f] 
`,

	detailHelpText: ` 
    该命令检查当前ossutil的版本与最新版本，输出两者的版本号，如果有更新版本，询问是否
    进行升级。如果指定了--force选项，则不询问，当有可用更新时，直接升级。

`,

	sampleText: ` 
    ossutil update
    ossutil update -f
`,
}

var specEnglishUpdate = SpecText{

	synopsisText: "Update ossutil",

	paramText: "[options]",

	syntaxText: ` 
    ossutil update [-f]
`,

	detailHelpText: ` 
    The command check version of current ossutil and get the latest version, output the 
    versions, if any updated version exists, the command ask you for upgrading. If --force 
    option is specified, the command upgrade without asking. 
`,

	sampleText: ` 
    ossutil update
    ossutil update -f
`,
}

// UpdateCommand is the command update ossutil
type UpdateCommand struct {
	command Command
}

var updateCommand = UpdateCommand{
	command: Command{
		name:        "update",
		nameAlias:   []string{""},
		minArgc:     0,
		maxArgc:     0,
		specChinese: specChineseUpdate,
		specEnglish: specEnglishUpdate,
		group:       GroupTypeAdditionalCommand,
		validOptionNames: []string{
			OptionForce,
			OptionRetryTimes,
			OptionLanguage,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionLogLevel,
		},
	},
}

// function for RewriteLoadConfiger interface
func (uc *UpdateCommand) rewriteLoadConfig(configFile string) error {
	// read config file, if error exist, do not print error
	var err error
	if uc.command.configOptions, err = LoadConfig(configFile); err != nil {
		uc.command.configOptions = OptionMapType{}
	}
	return nil
}

// function for FormatHelper interface
func (uc *UpdateCommand) formatHelpForWhole() string {
	return uc.command.formatHelpForWhole()
}

func (uc *UpdateCommand) formatIndependHelp() string {
	return uc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (uc *UpdateCommand) Init(args []string, options OptionMapType) error {
	return uc.command.Init(args, options, uc)
}

// RunCommand simulate inheritance, and polymorphism
func (uc *UpdateCommand) RunCommand() error {
	force, _ := GetBool(OptionForce, uc.command.options)
	language, _ := GetString(OptionLanguage, uc.command.options)
	language = strings.ToLower(language)

	// get lastest version
	version, err := uc.getLastestVersion()
	if err != nil {
		return fmt.Errorf("get lastest vsersion error, %s", err.Error())
	}

	if language == LEnglishLanguage {
		fmt.Printf("current version is: %s, the lastest version is: %s", vVersion, version)
	} else {
		fmt.Printf("当前版本为：%s，最新版本为：%s", vVersion, version)
	}

	// version is X.X.X
	// vVersion is vX.X.X
	if vVersion[0] < '0' || vVersion[0] > '9' {
		vVersion = vVersion[1:len(vVersion)]
	}
	if version == vVersion {
		if language == LEnglishLanguage {
			fmt.Println(", current version is the lastest version, no need to update.")
		} else {
			fmt.Println("，当前版本即为最新版本，无需更新。")
		}
		return nil
	}
	fmt.Println("")

	if !force {
		if language == LEnglishLanguage {
			fmt.Printf("sure to update ossutil(y or N)? ")
		} else {
			fmt.Printf("确定更新版本(y or N)? ")
		}

		var val string
		if _, err := fmt.Scanln(&val); err == nil && (strings.EqualFold(val, "yes") || strings.EqualFold(val, "y")) {
			return uc.updateVersion(version, language)
		}

		if language == LEnglishLanguage {
			fmt.Printf("operation is canceled.")
		} else {
			fmt.Println("操作取消。")
		}
	} else {
		return uc.updateVersion(version, language)
	}
	return nil
}

func (uc *UpdateCommand) getLastestVersion() (string, error) {
	if err := uc.anonymousGetToFileRetry(vUpdateBucket, updateVersionObject, updateTmpVersionFile); err != nil {
		return "", err
	}

	v, err := ioutil.ReadFile(updateTmpVersionFile)
	if err != nil {
		return "", err
	}
	versionStr := strings.TrimSpace(strings.Trim(string(v), "\n"))

	// get version list and sort
	sli := strings.Split(versionStr, "\n")
	vl := []string{}
	for _, vstr := range sli {
		vl = append(vl, strings.TrimSpace(strings.Trim(string(vstr), "\n")))
	}
	sort.Strings(vl)
	version := vl[len(vl)-1]

	os.Remove(updateTmpVersionFile)

	return version, nil
}

func (uc *UpdateCommand) anonymousGetToFileRetry(bucketName, objectName, filePath string) error {
	host := fmt.Sprintf("http://%s.%s/%s", bucketName, vUpdateEndpoint, objectName)
	retryTimes, _ := GetInt(OptionRetryTimes, uc.command.options)
	for i := 1; ; i++ {
		err := uc.ossAnonymousGetToFile(host, filePath)
		if err == nil {
			return err
		}
		if int64(i) >= retryTimes {
			return ObjectError{err, bucketName, objectName}
		}
	}
}

func (uc *UpdateCommand) ossAnonymousGetToFile(host, filePath string) error {
	response, err := http.Get(host)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	statusCode := response.StatusCode
	body, _ := ioutil.ReadAll(response.Body)
	if statusCode >= 300 {
		return fmt.Errorf(string(body))
	}

	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	defer fd.Close()
	if err != nil {
		return err
	}

	_, err = io.WriteString(fd, string(body))
	if err != nil {
		return err
	}
	return nil
}

func (uc *UpdateCommand) updateVersion(version, language string) error {
	// get binary path
	filePath, renameFilePath := getBinaryPath()

	// get binary mode
	f, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	mode := f.Mode()

	// rename the current binary to another one
	if err := os.Rename(filePath, renameFilePath); err != nil {
		return fmt.Errorf("update binary error, %s", err.Error())
	}

	// download the binary of the specified version
	if err := uc.getBinary(filePath, version); err != nil {
		uc.revertRename(filePath, renameFilePath)
		return fmt.Errorf("download binary of version: %s error, %s", version, err.Error())
	}

	if err := os.Chmod(filePath, mode); err != nil {
		uc.revertRename(filePath, renameFilePath)
		return fmt.Errorf("chmod binary error, %s", err.Error())
	}

	// remove the current one
	if runtime.GOOS != "windows" {
		if err := os.Remove(renameFilePath); err != nil {
			uc.revertRename(filePath, renameFilePath)
			return fmt.Errorf("remove old binary error, %s", err.Error())
		}
	}

	if language == LEnglishLanguage {
		fmt.Println("Update Success!")
	} else {
		fmt.Println("更新成功!")
	}
	return nil
}

func (uc *UpdateCommand) revertRename(filePath, renameFilePath string) error {
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}

	if err := os.Rename(renameFilePath, filePath); err != nil {
		return err
	}
	return nil
}

func (uc *UpdateCommand) getBinaryName() string {
	// get os type
	var object string
	switch runtime.GOOS {
	case "darwin":
		object = updateBinaryMac64
		switch runtime.GOARCH {
		case "386":
			object = updateBinaryMac32
		case "arm64":
			object = updateBinaryMacArm64
		}
	case "windows":
		object = updateBinaryWindow64
		if runtime.GOARCH == "386" {
			object = updateBinaryWindow32
		}
	default:
		object = updateBinaryLinux64
		switch runtime.GOARCH {
		case "386":
			object = updateBinaryLinux32
		case "arm":
			object = updateBinaryLinuxArm32
		case "arm64":
			object = updateBinaryLinuxArm64
		}
	}
	return object
}

func (uc *UpdateCommand) getBinary(filePath, version string) error {
	object := version + "/" + uc.getBinaryName()
	if err := uc.anonymousGetToFileRetry(vUpdateBucket, object, filePath); err != nil {
		return err
	}
	return nil
}
