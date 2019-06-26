// Copyright 1999-2019 Alibaba Group Holding Limited
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
	"fmt"
	"io"
	"os"

	"github.com/aliyun/aliyun-cli/cli"
	ini "gopkg.in/ini.v1"
)

func MigrateLegacyConfiguration(w io.Writer) *Configuration {
	path := hookGetHomePath(GetHomePath)() + "/.aliyuncli/credentials"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	result, err := MigrateCredentials(path)
	if err != nil {
		return nil
	}

	path = hookGetHomePath(GetHomePath)() + "/.aliyuncli/configure"
	err = MigrateConfigure(w, path, &result)
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
				Name:            profileName,
				Mode:            AK,
				AccessKeyId:     k1.String(),
				AccessKeySecret: k2.String(),
				OutputFormat:    "json",
			})
			// fmt.Printf(" %s/%s Done", MosaicString(k1.Text(), 3), MosaicString(k2.Text(), 3))
		} else {
			// fmt.Printf(" invalid! %v %v", e1, e2)
		}
	}
	return r, nil
}

func MigrateConfigure(w io.Writer, path string, config *Configuration) error {
	ini, err := ini.Load(path)
	if err != nil {
		cli.Errorf(w, " parse failed: %v\n", err)
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
