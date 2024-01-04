// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"fmt"
	"os"

	ini "gopkg.in/ini.v1"
)

func MigrateLegacyConfiguration() (conf *Configuration, err error) {
	path := hookGetHomePath(GetHomePath)() + "/.aliyuncli/credentials"
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		return
	}

	conf, migrateErr := MigrateCredentials(path)
	if migrateErr != nil {
		return
	}

	path = hookGetHomePath(GetHomePath)() + "/.aliyuncli/configure"
	mfErr := MigrateConfigure(path, conf)
	if mfErr != nil {
		return
	}

	return
}

func MigrateCredentials(path string) (conf *Configuration, err error) {
	conf = &Configuration{}
	ini, err := ini.Load(path)
	if err != nil {
		err = fmt.Errorf(" parse failed: %v", err)
		return
	}

	for _, section := range ini.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		profileName := section.Name()
		if section.Name() == "default" {
			conf.CurrentProfile = "default"
		} else {
			profileName = section.Name()[len("profile "):]
			if conf.CurrentProfile != "default" {
				conf.CurrentProfile = profileName
			}
		}
		// fmt.Printf("\n  %s:", profileName)
		k1, e1 := section.GetKey("aliyun_access_key_id")
		k2, e2 := section.GetKey("aliyun_access_key_secret")
		if e1 == nil && e2 == nil {
			conf.Profiles = append(conf.Profiles, Profile{
				Name:            profileName,
				Mode:            AK,
				AccessKeyId:     k1.String(),
				AccessKeySecret: k2.String(),
				OutputFormat:    "json",
			})
		}
	}
	return
}

func MigrateConfigure(path string, conf *Configuration) (err error) {
	ini, err := ini.Load(path)
	if err != nil {
		err = fmt.Errorf("parse failed: %s", err)
		return
	}

	for _, section := range ini.Sections() {
		profile, ok := conf.GetProfile(section.Name())
		if !ok {
			continue
		}

		r, err := section.GetKey("region")
		if err == nil {
			profile.RegionId = r.String()
			conf.PutProfile(profile)
		}
	}

	return nil
}
