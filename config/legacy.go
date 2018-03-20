/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"os"
	"github.com/aliyun/aliyun-cli/cli"
	"gopkg.in/ini.v1"
	"fmt"
)

func MigrateLegacyConfiguration() (*Configuration) {
	path := GetHomePath() + "/.aliyuncli/credentials"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	result, err := MigrateCredentials(path)
	if err != nil {
		return nil
	}

	path = GetHomePath() + "/.aliyuncli/configure"
	err = MigrateConfigure(path, &result)
	if err != nil {
		return nil
	}

	return &result
}

func MigrateCredentials(path string) (Configuration, error) {
	r := Configuration{}
	ini, err := ini.Load(path)
	if err != nil {
		return r, fmt.Errorf(" parse failed: %v\n", err)
	}

	for _, section := range ini.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		profileName := section.Name()
		if section.Name() == "default" {
			r.CurrentProfile = "default"
		} else {
			profileName = section.Name()[len("profile "):]
			if r.CurrentProfile != "default" {
				r.CurrentProfile = profileName
			}
		}
		// fmt.Printf("\n  %s:", profileName)
		k1, e1 := section.GetKey("aliyun_access_key_id")
		k2, e2 := section.GetKey("aliyun_access_key_secret")
		if e1 == nil && e2 == nil {
			r.Profiles = append(r.Profiles, Profile{
				Name: profileName,
				Mode: AK,
				AccessKeyId: k1.String(),
				AccessKeySecret: k2.String(),
				OutputFormat: "json",
			})
			// fmt.Printf(" %s/%s Done", MosaicString(k1.Text(), 3), MosaicString(k2.Text(), 3))
		} else {
			// fmt.Printf(" invalid! %v %v", e1, e2)
		}
	}
	return r, nil
}

func MigrateConfigure(path string, config *Configuration) error {
	ini, err := ini.Load(path)
	if err != nil {
		cli.Errorf(" parse failed: %v\n", err)
		return err
	}

	for _, section := range ini.Sections() {
		profile, ok := config.GetProfile(section.Name())
		if !ok {
			continue
		}

		r, err := section.GetKey("region")
		if err == nil {
			profile.RegionId = r.String()
			config.PutProfile(profile)
		}
	}
	return nil
}


