/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"io"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewConfigureDeleteCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "delete",
		Usage: "delete --profile <profileName>",
		Short: i18n.T("list all config profile", "列出所有配置集"),
		Run: func(c *cli.Context, args []string) error {
			profileName, ok := ProfileFlag(c.Flags()).GetValue()
			if !ok {
				cli.Errorf(c.Writer(), "missing --profile <profileName>\n")
				cli.Noticef(c.Writer(), "\nusage:\n  aliyun configure delete --profile <profileName>\n")
				return nil
			}
			doConfigureDelete(c.Writer(), profileName)
			return nil
		},
	}
	return cmd
}

func doConfigureDelete(w io.Writer, profileName string) {
	conf, err := hookLoadConfiguration(LoadConfiguration)(GetConfigPath()+"/"+configFile, w)
	if err != nil {
		cli.Errorf(w, "ERROR: load configure failed: %v\n", err)
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
		cli.Errorf(w, "Error: configuration profile `%s` not found\n", profileName)
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

	err = hookSaveConfiguration(SaveConfiguration)(conf)
	if err != nil {
		cli.Errorf(w, "Error: save configuration failed %s\n", err)
	}
}
