// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

var hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
	return fn
}

var hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
	return fn
}

func loadConfiguration() (*Configuration, error) {
	return hookLoadConfiguration(LoadConfiguration)(GetConfigPath() + "/" + configFile)
}

func NewConfigureCommand() *cli.Command {

	c := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName|ChainableRamRoleArn} --profile <profileName>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			profileName, _ := ProfileFlag(ctx.Flags()).GetValue()
			mode, _ := ModeFlag(ctx.Flags()).GetValue()
			return doConfigure(ctx, profileName, mode)
		},
	}

	c.AddSubCommand(NewConfigureGetCommand())
	c.AddSubCommand(NewConfigureSetCommand())
	c.AddSubCommand(NewConfigureListCommand())
	c.AddSubCommand(NewConfigureDeleteCommand())
	return c
}

func doConfigure(ctx *cli.Context, profileName string, mode string) error {
	w := ctx.Writer()

	conf, err := loadConfiguration()
	if err != nil {
		return err
	}

	if profileName == "" {
		if conf.CurrentProfile == "" {
			profileName = "default"
		} else {
			profileName = conf.CurrentProfile
			originMode := string(conf.GetCurrentProfile(ctx).Mode)
			if mode == "" {
				mode = originMode
			} else if mode != originMode {
				cli.Printf(w, "Warning: You are changing the authentication type of profile '%s' from '%s' to '%s'\n", profileName, originMode, mode)
			}
		}
	}
	if mode == "" {
		mode = "AK"
	}
	cp, ok := conf.GetProfile(profileName)
	if !ok {
		cp = conf.NewProfile(profileName)
	}

	cli.Printf(w, "Configuring profile '%s' in '%s' authenticate mode...\n", profileName, mode)

	if mode != "" {
		switch AuthenticateMode(mode) {
		case AK:
			cp.Mode = AK
			configureAK(w, &cp)
		case StsToken:
			cp.Mode = StsToken
			configureStsToken(w, &cp)
		case RamRoleArn:
			cp.Mode = RamRoleArn
			configureRamRoleArn(w, &cp)
		case EcsRamRole:
			cp.Mode = EcsRamRole
			configureEcsRamRole(w, &cp)
		case RamRoleArnWithEcs:
			cp.Mode = RamRoleArnWithEcs
			configureRamRoleArnWithEcs(w, &cp)
		case ChainableRamRoleArn:
			cp.Mode = ChainableRamRoleArn
			configureChainableRamRoleArn(w, &cp)
		case RsaKeyPair:
			cp.Mode = RsaKeyPair
			configureRsaKeyPair(w, &cp)
		case External:
			cp.Mode = External
			configureExternal(w, &cp)
		case CredentialsURI:
			cp.Mode = CredentialsURI
			configureCredentialsURI(w, &cp)
		default:
			return fmt.Errorf("unexcepted authenticate mode: %s", mode)
		}
	} else {
		configureAK(w, &cp)
	}

	//
	// configure common
	cli.Printf(w, "Default Region Id [%s]: ", cp.RegionId)
	cp.RegionId = ReadInput(cp.RegionId)
	cli.Printf(w, "Default Output Format [%s]: json (Only support json)\n", cp.OutputFormat)

	// cp.OutputFormat = ReadInput(cp.OutputFormat)
	cp.OutputFormat = "json"

	cli.Printf(w, "Default Language [zh|en] %s: ", cp.Language)

	cp.Language = ReadInput(cp.Language)
	if cp.Language != "zh" && cp.Language != "en" {
		cp.Language = "en"
	}

	//fmt.Printf("User site: [china|international|japan] %s", cp.Site)
	//cp.Site = ReadInput(cp.Site)

	cli.Printf(w, "Saving profile[%s] ...", profileName)

	conf.PutProfile(cp)
	conf.CurrentProfile = cp.Name
	err = hookSaveConfiguration(SaveConfiguration)(conf)
	// cp 要在下文的 DoHello 中使用，所以 需要建立 parent 的关系
	cp.parent = conf

	if err != nil {
		return err
	}
	cli.Printf(w, "Done.\n")

	DoHello(ctx, &cp)
	return nil
}

func configureAK(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Access Key Id [%s]: ", MosaicString(cp.AccessKeyId, 3))
	cp.AccessKeyId = ReadInput(cp.AccessKeyId)
	cli.Printf(w, "Access Key Secret [%s]: ", MosaicString(cp.AccessKeySecret, 3))
	cp.AccessKeySecret = ReadInput(cp.AccessKeySecret)
	return nil
}

func configureStsToken(w io.Writer, cp *Profile) error {
	err := configureAK(w, cp)
	if err != nil {
		return err
	}
	cli.Printf(w, "Sts Token [%s]: ", cp.StsToken)
	cp.StsToken = ReadInput(cp.StsToken)
	return nil
}

func configureRamRoleArn(w io.Writer, cp *Profile) error {
	err := configureAK(w, cp)
	if err != nil {
		return err
	}
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureEcsRamRole(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Ecs Ram Role [%s]: ", cp.RamRoleName)
	cp.RamRoleName = ReadInput(cp.RamRoleName)
	return nil
}

func configureRamRoleArnWithEcs(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Ecs Ram Role [%s]: ", cp.RamRoleName)
	cp.RamRoleName = ReadInput(cp.RamRoleName)
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureChainableRamRoleArn(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Source Profile [%s]: ", cp.SourceProfile)
	cp.SourceProfile = ReadInput(cp.SourceProfile)
	cli.Printf(w, "Sts Region [%s]: ", cp.StsRegion)
	cp.StsRegion = ReadInput(cp.StsRegion)
	cli.Printf(w, "Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	cli.Printf(w, "Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	if cp.ExpiredSeconds == 0 {
		cp.ExpiredSeconds = 900
	}
	cli.Printf(w, "Expired Seconds [%v]: ", cp.ExpiredSeconds)
	cp.ExpiredSeconds, _ = strconv.Atoi(ReadInput(strconv.Itoa(cp.ExpiredSeconds)))
	return nil
}

func configureRsaKeyPair(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Rsa Private Key File: ")
	keyFile := ReadInput("")
	buf, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read key file %s failed %v", keyFile, err)
	}
	cp.PrivateKey = string(buf)
	cli.Printf(w, "Rsa Key Pair Name: ")
	cp.KeyPairName = ReadInput("")
	cp.ExpiredSeconds = 900
	return nil
}

func configureExternal(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Process Command [%s]: ", cp.ProcessCommand)
	cp.ProcessCommand = ReadInput(cp.ProcessCommand)
	return nil
}

func configureCredentialsURI(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Credentials URI [%s]: ", cp.CredentialsURI)
	cp.CredentialsURI = ReadInput(cp.CredentialsURI)
	return nil
}

func ReadInput(defaultValue string) string {
	var s string
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		s = scanner.Text()
	}
	if s == "" {
		return defaultValue
	}
	return s
}

func MosaicString(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return strings.Repeat("*", r) + s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}

func GetLastChars(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}
