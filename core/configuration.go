package core

import (
	"fmt"
	"io/ioutil"
	"encoding/json"
	"runtime"
	"os"
)

const (
	configPath = "/.aliyun"
	configFile = "config.json"
	DefaultConfigProfile = "default"
)

type Profile struct {
	Name string				`json:"name"`
	AccessKeyId string		`json:"access_key_id"`
	AccessKeySecret string	`json:"access_key_secret"`
	RegionId string 		`json:"region_id"`
	OutputFormat string 	`json:"output_format"`
	Language string			`json:"language"`
}

func NewProfile(name string) (Profile) {
	return Profile {
		Name: name,
		OutputFormat: "json",
		Language: "en-US",
	}
}

type Configuration struct {
	CurrentProfile string	`json:"current"`
	Profiles []Profile		`json:"profiles"`
}

func NewConfiguration() (Configuration) {
	return Configuration{
		CurrentProfile: DefaultConfigProfile,
		Profiles: []Profile{
			NewProfile(DefaultConfigProfile),
		},
	}
}

func (c *Configuration) NewProfile(pn string) (Profile) {
	p, ok := c.GetProfile(pn)
	if !ok {
		p = NewProfile(pn)
		c.PutProfile(p)
	}
	return p
}

func (c *Configuration) GetProfile(pn string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.Name == pn {
			return p, true
		}
	}
	return Profile{}, false
}

func (c *Configuration) GetCurrentProfile() (Profile) {
	p, _ := c.GetProfile(c.CurrentProfile)
	return p
}


func (c *Configuration) PutProfile(profile Profile) {
	for i, p := range c.Profiles {
		if p.Name == profile.Name {
			c.Profiles[i] = profile
			return
		}
	}
	c.Profiles = append(c.Profiles, profile)
}

func LoadConfiguration() (Configuration, error) {
	path := GetConfigPath() + "/" + configFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewConfiguration(), nil
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return NewConfiguration(), fmt.Errorf("reading config from '%s' failed %v", path, err)
	}

	return NewConfigFromBytes(bytes)
}

func SaveConfiguration(config Configuration) (error) {
	// fmt.Printf("conf %v\n", config)
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	path := GetConfigPath() + "/" + configFile
	err = ioutil.WriteFile(path, bytes, 0755)
	if err != nil {
		return err
	}
	return nil
}

func NewConfigFromBytes(bytes []byte) (Configuration, error) {
	var conf Configuration
	err := json.Unmarshal(bytes, &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}

func GetConfigPath() string {
	path := GetHomePath() + configPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			panic(err)
		}
	}
	return path
}

func GetHomePath() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
