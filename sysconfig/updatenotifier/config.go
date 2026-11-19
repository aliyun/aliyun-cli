// Package updatenotifier stores the user-wide update notification preference.
package updatenotifier

import (
	"encoding/json"
	"github.com/aliyun/aliyun-cli/v3/internal/localstate"
	"os"
	"path/filepath"
	"strings"
)

const DisableEnv = "ALIBABA_CLOUD_CLI_NO_UPDATE_NOTIFIER"
const ConfigFileName = "update-notifier.json"

type Config struct {
	Enabled bool `json:"enabled"`
}

func EnvironmentDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(DisableEnv))) {
	case "1", "true", "yes":
		return true
	}
	return false
}
func Load(dir string) (*Config, error) {
	c := &Config{Enabled: true}
	data, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}
func Enabled(dir string) bool {
	if EnvironmentDisabled() {
		return false
	}
	c, err := Load(dir)
	return err == nil && c.Enabled
}
func Save(dir string, enabled bool) error {
	data, err := json.MarshalIndent(Config{Enabled: enabled}, "", "  ")
	if err != nil {
		return err
	}
	return localstate.Write(filepath.Join(dir, ConfigFileName), append(data, '\n'))
}
