package lib

import (
	"fmt"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseListCloudBox = SpecText{

	synopsisText: "列举云盒信息",

	paramText: "[options]",

	syntaxText: ` 
    ossutil lcb [-e endpoint]
`,

	detailHelpText: `
    该命令列举云盒的详细信息
`,

	sampleText: ` 
    1) ossutil lcb --sign-version v4 --region cn-hangzhou --cloudbox-id cb-abcdef
`,
}

var specEnglishListCloudBox = SpecText{

	synopsisText: "List cloud box information",

	paramText: "[options]",

	syntaxText: ` 
    ossutil lcb [-e endpoint] 
`,

	detailHelpText: ` 
    This command lists cloud box information
`,

	sampleText: ` 
    1) ossutil lcb --sign-version v4 --region cn-hangzhou --cloudbox-id cb-abcdef
`,
}

// LcbCommand is the command list region buckets or objects
type LcbCommand struct {
	command Command
}

var lcbCommand = LcbCommand{
	command: Command{
		name:        "lcb",
		nameAlias:   []string{"lcb"},
		minArgc:     0,
		maxArgc:     1,
		specChinese: specChineseListCloudBox,
		specEnglish: specEnglishListCloudBox,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionRetryTimes,
			OptionLogLevel,
			OptionPassword,
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionReadTimeout,
			OptionConnectTimeout,
			OptionSTSRegion,
			OptionSkipVerifyCert,
			OptionUserAgent,
			OptionRegion,
			OptionSignVersion,
			OptionLimitedNum,
			OptionMarker,
			OptionCloudBoxID,
			OptionForcePathStyle,
		},
	},
}

// function for FormatHelper interface
func (lc *LcbCommand) formatHelpForWhole() string {
	return lc.command.formatHelpForWhole()
}

func (lc *LcbCommand) formatIndependHelp() string {
	return lc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (lc *LcbCommand) Init(args []string, options OptionMapType) error {
	return lc.command.Init(args, options, lc)
}

// RunCommand simulate inheritance, and polymorphism
func (lc *LcbCommand) RunCommand() error {
	prefix := ""
	if len(lc.command.args) > 0 {
		cloudURL, err := CloudURLFromString(lc.command.args[0], "")
		if err != nil {
			return err
		}
		prefix = cloudURL.bucket
	}

	limitedNum, _ := GetInt(OptionLimitedNum, lc.command.options)
	vmarker, _ := GetString(OptionMarker, lc.command.options)
	if vmarker, err := lc.command.getRawMarker(vmarker); err != nil {
		return fmt.Errorf("invalid marker: %s, marker is not url encoded, %s", vmarker, err.Error())
	}

	var num int64
	num = 0

	client, err := lc.command.ossClient("")
	if err != nil {
		return err
	}

	// list all cloudbox
	pre := oss.Prefix(prefix)
	marker := oss.Marker(vmarker)
	for limitedNum < 0 || num < limitedNum {
		lcr, err := lc.ossListCloudBoxesRetry(client, pre, marker)
		if err != nil {
			return err
		}
		pre = oss.Prefix(lcr.Prefix)
		marker = oss.Marker(lcr.NextMarker)
		for _, box := range lcr.CloudBoxes {
			if limitedNum >= 0 && num >= limitedNum {
				break
			}
			fmt.Printf("%-15s:%d\n", "No", num)
			fmt.Printf("%-15s:%s\n", "Id", box.ID)
			fmt.Printf("%-15s:%s\n", "Name", box.Name)
			fmt.Printf("%-15s:%s\n", "Owner", lcr.Owner)
			fmt.Printf("%-15s:%s\n", "Region", box.Region)
			fmt.Printf("%-15s:%s\n", "ControlEndpoint", box.ControlEndpoint)
			fmt.Printf("%-15s:%s\n", "DataEndpoint", box.DataEndpoint)
			fmt.Printf("----------------------------------------------------------------------\n")
			num++
		}
		if !lcr.IsTruncated {
			break
		}
	}
	return nil
}

func (lc *LcbCommand) ossListCloudBoxesRetry(client *oss.Client, options ...oss.Option) (oss.ListCloudBoxResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, lc.command.options)
	for i := 1; ; i++ {
		lbr, err := client.ListCloudBoxes(options...)
		if err == nil || int64(i) >= retryTimes {
			return lbr, err
		}
	}
}
