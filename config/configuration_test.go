/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfiguration(t *testing.T) {
	excf := Configuration{
		CurrentProfile: DefaultConfigProfileName,
		Profiles: []Profile{
			NewProfile(DefaultConfigProfileName),
		},
	}
	cf := NewConfiguration()
	assert.Equal(t, excf, cf)
}

func TestCFNewProfile(t *testing.T) {
	cf := Configuration{
		CurrentProfile: "",
	}
	assert.Len(t, cf.Profiles, 0)
	p := cf.NewProfile("default")
	assert.Len(t, cf.Profiles, 1)
	exp := Profile{
		Name:         "default",
		Mode:         AK,
		OutputFormat: "json",
		Language:     "en",
	}
	assert.Equal(t, exp, p)
}

func TestConfiguration(t *testing.T) {
	cf := NewConfiguration()
	//GetProfile
	p, ok := cf.GetProfile("hh")
	assert.False(t, ok)
	assert.Equal(t, Profile{Name: "hh"}, p)
	p, ok = cf.GetProfile("default")
	assert.Equal(t, Profile{Name: "default", Mode: AK, OutputFormat: "json", Language: "en"}, p)

	//PutProfile
	assert.Len(t, cf.Profiles, 1)
	cf.PutProfile(Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"}, cf.Profiles[1])
	cf.PutProfile(Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, cf.Profiles[1])

	//GetCurrentProfile

}
