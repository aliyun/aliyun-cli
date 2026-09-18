package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Preview capabilities are deliberately operation-specific. Never call a legacy
// executor to discover whether it supports a dry run: it may already write.
type ossPreview struct {
	SchemaVersion string        `json:"schema_version"`
	Mode          string        `json:"mode"`
	Command       string        `json:"command"`
	Complete      bool          `json:"complete"`
	SideEffects   string        `json:"side_effects"`
	Items         []previewItem `json:"items"`
	Checks        []string      `json:"checks"`
	Limitations   []string      `json:"limitations"`
}
type previewItem struct {
	Action    string `json:"action"`
	Source    string `json:"source,omitempty"`
	Target    string `json:"target"`
	VersionID string `json:"version_id,omitempty"`
	Exists    *bool  `json:"target_exists,omitempty"`
	ETag      string `json:"target_etag,omitempty"`
}

func previewMode(ctx *cli.Context) (string, error) {
	mode := ""
	for _, name := range []string{"cli-validate", "cli-plan"} {
		if f := ctx.Flags().Get(name); f != nil && f.IsAssigned() {
			if mode != "" {
				return "", fmt.Errorf("--cli-validate and --cli-plan are mutually exclusive")
			}
			mode = strings.TrimPrefix(name, "cli-")
		}
	}
	return mode, nil
}

func preparePreview(cmd *Command, args, tokens []string, options OptionMapType, mode string) (*ossPreview, error) {
	if cmd == nil {
		return nil, fmt.Errorf("preview requires an OSS subcommand")
	}
	p := &ossPreview{Limitations: []string{"does not verify write permissions", "metadata is a snapshot, not a lock or an execution token", "no cloud or local writes are executed"}, SchemaVersion: "1", Mode: mode, Command: cmd.name, Complete: true, SideEffects: "none", Items: []previewItem{}, Checks: []string{"argument_count", "option_types", "uri_structure"}}
	// Allow configuration/transport flags, but reject every business flag whose
	// semantics have not been implemented here (including filters and recursion).
	allowed := map[string]bool{}
	for _, name := range []string{OptionUserAgent, OptionEndpoint, OptionAccessKeyID, OptionAccessKeySecret, OptionSTSToken, OptionRegion, OptionSignVersion, OptionRetryTimes, OptionReadTimeout, OptionConnectTimeout, OptionConfigFile, OptionForce, OptionForcePathStyle, OptionSkipVerifyCert, OptionProxyHost, OptionProxyUser, OptionProxyPwd} {
		allowed[OptionMap[name].nameAlias] = true
		if OptionMap[name].name != "" {
			allowed[OptionMap[name].name] = true
		}
	}
	if cmd.name == "rm" {
		allowed[OptionMap[OptionVersionId].nameAlias] = true
	}
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "--" {
			break
		}
		key, _, inline := strings.Cut(tokens[i], "=")
		if !strings.HasPrefix(key, "-") || key == "-" {
			continue
		}
		if !allowed[key] {
			return nil, fmt.Errorf("oss %s preview does not support %s; no operation was executed", cmd.name, key)
		}
		for _, opt := range OptionMap {
			if (key == opt.nameAlias || key == opt.name) && opt.optionType != OptionTypeFlagTrue && !inline {
				i++
				break
			}
		}
	}
	object := func(s string) (CloudURL, error) {
		if !strings.HasPrefix(s, SchemePrefix) {
			return CloudURL{}, fmt.Errorf("preview requires an explicit oss://bucket/key URI")
		}
		u, err := ObjectURLFromString(s, "")
		if err != nil {
			return u, err
		}
		if err = u.checkObjectPrefix(); err != nil {
			return u, err
		}
		if strings.HasSuffix(u.object, "/") {
			return u, fmt.Errorf("preview requires a full object key without a trailing slash")
		}
		return u, nil
	}
	switch cmd.name {
	case "cp":
		if len(args) != 2 {
			return nil, fmt.Errorf("upload preview requires exactly one source and one target")
		}
		if strings.Contains(args[0], "://") {
			return nil, fmt.Errorf("oss cp preview currently supports only a single local file upload to an explicit object key")
		}
		if _, err := object(args[1]); err != nil {
			return nil, err
		}
		info, err := os.Stat(args[0])
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("upload preview requires a regular local file")
		}
		p.Checks = append(p.Checks, "local_source_is_regular_file")
		p.Items = append(p.Items, previewItem{Action: "upload", Source: args[0], Target: args[1]})
	case "rm":
		if len(args) != 1 {
			return nil, fmt.Errorf("delete preview requires exactly one target")
		}
		if _, err := object(args[0]); err != nil {
			return nil, err
		}
		version, _ := GetString(OptionVersionId, options)
		p.Items = append(p.Items, previewItem{Action: "delete", Target: args[0], VersionID: version})
	default:
		return nil, fmt.Errorf("oss %s does not support validate/plan; supported operations: cp single local file upload, rm single object (optional version-id); no operation was executed", cmd.name)
	}
	return p, nil
}

func (p *ossPreview) run(ctx *cli.Context, cmd Command, parsed, resolved OptionMapType) error {
	if p.Mode == "plan" {
		if activeMachine != nil {
			activeMachine.phase = "plan"
		}
		// Initialize only the shared configuration, never the command's executor.
		for k, v := range resolved {
			parsed[k] = v
		}
		if err := cmd.Init(cmd.args, parsed, nil); err != nil {
			return err
		}
		for i := range p.Items {
			item := &p.Items[i]
			u, _ := ObjectURLFromString(item.Target, "")
			bucket, err := cmd.ossBucket(u.bucket)
			if err != nil {
				return err
			}
			opts := []oss.Option{}
			if item.VersionID != "" {
				opts = append(opts, oss.VersionId(item.VersionID))
			}
			headers, err := cmd.ossGetObjectStatRetry(bucket, u.object, opts...)
			exists := true
			if err != nil {
				var se oss.ServiceError
				if errors.As(err, &se) && se.StatusCode == 404 && (se.Code == "NoSuchKey" || se.Code == "NoSuchVersion") {
					exists = false
				} else {
					return err
				}
			}
			item.Exists = &exists
			item.ETag = headers.Get("ETag")
		}
		p.Checks = append(p.Checks, "remote_target_metadata")
	}
	return json.NewEncoder(ctx.Stdout()).Encode(p)
}
