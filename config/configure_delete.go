/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewConfigureDeleteCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "delete",
		Usage: "delete --profile <profileName>",
		Short: i18n.T("list all config profile", "列出所有配置集"),
		Run: func(c *cli.Context, args []string) error {
			profileName, ok := ProfileFlag.GetValue()
			if !ok {
				cli.Errorf("missing --profile <profileName>\n")
				cli.Noticef("\nusage:\n  aliyun configure delete --profile <profileName>\n")
				return nil
			}
			doConfigureDelete(profileName)
			return nil
		},
	}
	cmd.Flags().Add(ProfileFlag)
	return cmd
}

func doConfigureDelete(profileName string) {
	conf, err := LoadConfiguration()
	if err != nil {
		cli.Errorf("ERROR: load configure failed: %v\n", err)
	}
	deleted := false
	r := make([]Profile, 0)
	for _, p := range conf.Profiles {
		if p.Name != profileName {
			r = append(r, p)
		} else {
			deleted = true
		}
	}

	if !deleted {
		cli.Errorf("Error: configuration profile `%s` not found\n", profileName)
		return
	}

	conf.Profiles = r
	if conf.CurrentProfile == profileName {
		if len(conf.Profiles) > 0 {
			conf.CurrentProfile = conf.Profiles[0].Name
		} else {
			conf.CurrentProfile = DefaultConfigProfileName
		}
	}

	err = SaveConfiguration(conf)
	if err != nil {
		cli.Errorf("Error: save configuration failed %s\n", err)
	}
}
