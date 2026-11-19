package upgrade

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"golang.org/x/mod/semver"
)

// UpdateResult describes availability, never an instruction to install automatically.
type UpdateResult struct {
	Current         string    `json:"current"`
	Latest          string    `json:"latest,omitempty"`
	Target          string    `json:"target,omitempty"`
	UpdateAvailable bool      `json:"update_available"`
	Status          string    `json:"status"`
	Command         string    `json:"command"`
	CheckedAt       time.Time `json:"checked_at"`
}

func CheckUpdate(ctx context.Context, current string) (UpdateResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	r := UpdateResult{Current: current, Status: "failed", Command: "aliyun upgrade", CheckedAt: time.Now().UTC()}
	pointer, err := fetchVersionFromOSSContext(ctx)
	if err != nil {
		return r, fmt.Errorf("CLI version check failed")
	}
	latest, valid := canonicalUpgradeVersion(pointer)
	if !valid || semver.Prerelease("v"+latest) != "" {
		return r, fmt.Errorf("invalid stable CLI release version")
	}
	r.Latest = latest
	versions, present, err := fetchVersionListContext(ctx)
	if err != nil {
		return r, fmt.Errorf("CLI release catalog check failed")
	}
	if _, ok := canonicalUpgradeVersion(current); !ok || strings.Contains(strings.ToLower(current), "dev") || strings.HasPrefix(current, "0.0.") {
		r.Status = "development_build"
		return r, nil
	}
	r.Target = selectUpgradeVersion(current, latest, clientSignatureVersion, versions, present)
	r.UpdateAvailable = r.Target != ""
	r.Status = "up_to_date"
	if isNewer(current, latest) {
		r.Status = "no_compatible_release"
		if r.UpdateAvailable {
			r.Status = "update_available"
		}
	}
	return r, nil
}

// CheckRunner is supplied by the host to include installed plugins without
// introducing a dependency from the upgrader back into the notification service.
type CheckRunner func(*cli.Context) error
