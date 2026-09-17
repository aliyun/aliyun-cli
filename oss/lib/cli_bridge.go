package lib

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// ParseAndRunCommandFunc 定义ParseAndRunCommand函数类型
type ParseAndRunCommandFunc func() error

// parseAndRunCommandImpl 是真正的ParseAndRunCommand实现，可以被测试代码替换
var parseAndRunCommandImpl ParseAndRunCommandFunc = ParseAndRunCommand

func NewOssCommand() *cli.Command {
	result := &cli.Command{
		Name:   "oss",
		Usage:  "oss [command] [args...] [options...]",
		Hidden: false,
		Short:  i18n.T("Object Storage Service(deprecated, use aliyun ossutil instead)", "阿里云OSS对象存储（废弃，请使用aliyun ossutil）"),
	}

	cmds := []Command{
		helpCommand.command,
		configCommand.command,
		makeBucketCommand.command,
		listCommand.command,
		removeCommand.command,
		statCommand.command,
		setACLCommand.command,
		setMetaCommand.command,
		copyCommand.command,
		restoreCommand.command,
		createSymlinkCommand.command,
		readSymlinkCommand.command,
		signURLCommand.command,
		hashCommand.command,
		updateCommand.command,
		probeCommand.command,
		mkdirCommand.command,
		corsCommand.command,
		bucketLogCommand.command,
		bucketRefererCommand.command,
		listPartCommand.command,
		allPartSizeCommand.command,
		appendFileCommand.command,
		catCommand.command,
		bucketTagCommand.command,
		bucketEncryptionCommand.command,
		corsOptionsCommand.command,
		bucketLifeCycleCommand.command,
		bucketWebsiteCommand.command,
		bucketQosCommand.command,
		userQosCommand.command,
		bucketVersioningCommand.command,
		duSizeCommand.command,
		bucketPolicyCommand.command,
		requestPaymentCommand.command,
		objectTagCommand.command,
		bucketInventoryCommand.command,
		revertCommand.command,
		syncCommand.command,
		wormCommand.command,
		lrbCommand.command,
		replicationCommand.command,
		bucketCnameCommand.command,
		lcbCommand.command,
		bucketAccessMonitorCommand.command,
		bucketResourceGroupCommand.command,
	}

	result.Help = printHostOSSHelp
	addMachineFlags(result.Flags())
	for _, cmd := range cmds {
		result.AddSubCommand(NewCommandBridge(cmd))
	}
	return result
}

func NewCommandBridge(cmd Command) *cli.Command {

	result := &cli.Command{
		Name:     cmd.name,
		Usage:    cmd.name + " " + cmd.specEnglish.paramText,
		Hidden:   cmd.name == "update",
		Help:     printHostOSSHelp,
		Short:    i18n.T(cmd.specEnglish.synopsisText, cmd.specChinese.synopsisText),
		Long:     i18n.T(cmd.specEnglish.detailHelpText, cmd.specChinese.detailHelpText),
		KeepArgs: true,
		RawArgs:  true,
		Run: func(ctx *cli.Context, args []string) error {
			return parseAndRunCommandFromCli(ctx, args, &cmd)
		},
	}

	configFlags := cli.NewFlagSet()
	config.AddFlags(configFlags)
	for _, flag := range configFlags.Flags() {
		for _, name := range cmd.validOptionNames {
			opt := OptionMap[name]
			if opt.nameAlias == "--"+flag.Name && opt.name != "" {
				flag.Shorthand = rune(opt.name[1])
			}
		}
		result.Flags().Add(flag)
	}

	for _, s := range cmd.validOptionNames {
		opt, ok := OptionMap[s]
		if !ok {
			continue
		}
		name := opt.nameAlias[2:]

		shorthand := rune(0)
		if len(opt.name) > 0 {
			shorthand = rune(opt.name[1])
		}

		if result.Flags().Get(name) == nil {
			assignedMode := cli.AssignedOnce
			switch opt.optionType {
			case OptionTypeFlagTrue:
				assignedMode = cli.AssignedNone
			case OptionTypeStrings:
				assignedMode = cli.AssignedRepeatable
			}
			if s == OptionInclude || s == OptionExclude {
				assignedMode = cli.AssignedRepeatable
			}
			result.Flags().Add(&cli.Flag{
				Name:         name,
				Shorthand:    shorthand,
				Short:        i18n.T(opt.helpEnglish, opt.helpChinese),
				AssignedMode: assignedMode,
			})
		}
	}
	addMachineFlags(result.Flags())
	return result
}

// buildOssEndpoint builds the regional OSS endpoint.
// When endpointType is "vpc", use the internal endpoint (oss-{region}-internal.aliyuncs.com).
func buildOssEndpoint(region, endpointType string) string {
	if strings.EqualFold(strings.TrimSpace(endpointType), "vpc") {
		return "oss-" + region + "-internal.aliyuncs.com"
	}
	return "oss-" + region + ".aliyuncs.com"
}

func getArgValue(args []string, name string) (string, bool) {
	value, found := "", false
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		key, v, inline := strings.Cut(args[i], "=")
		if key == name {
			if inline {
				value, found = v, true
			} else if i+1 < len(args) {
				i++
				value, found = args[i], true
			}
			continue
		}
		for _, opt := range OptionMap {
			if (key == opt.nameAlias || key == opt.name) && opt.optionType != OptionTypeFlagTrue && !inline {
				i++
				break
			}
		}
	}
	return value, found
}

// ossKnownOptionNames returns long/short option names understood by OSS goopt (OptionMap).
func ossKnownOptionNames() map[string]struct{} {
	m := make(map[string]struct{}, len(OptionMap)*2)
	for _, opt := range OptionMap {
		if opt.nameAlias != "" {
			m[opt.nameAlias] = struct{}{}
		}
		if opt.name != "" {
			m[opt.name] = struct{}{}
		}
	}
	return m
}

// stripCliOnlyFlagsFromArgs removes aliyun CLI config flags that OSS goopt does not
// understand (e.g. --profile / -p), while keeping flags shared with OSS
// (e.g. --endpoint, --region). Keeps KeepArgs forwarding from breaking on
// trailing global flags. Supports --name value, --name=value, and short forms.
func stripCliOnlyFlagsFromArgs(args []string) []string {
	configFS := cli.NewFlagSet()
	config.AddFlags(configFS)
	ossKnown := ossKnownOptionNames()
	hostOnly := make(map[string]bool)
	for _, f := range configFS.Flags() {
		names := []string{"--" + f.Name}
		for _, alias := range f.Aliases {
			names = append(names, "--"+alias)
		}
		if f.Shorthand != 0 {
			names = append(names, "-"+string(f.Shorthand))
		}
		for _, name := range names {
			if _, shared := ossKnown[name]; !shared {
				hostOnly[name] = f.AssignedMode != cli.AssignedNone
			}
		}
	}
	for _, name := range []string{"--cli-ai-mode", "--no-cli-ai-mode", "--cli-non-interactive", "--cli-validate", "--cli-plan"} {
		hostOnly[name] = false
	}
	for _, name := range []string{"--cli-output", "--cli-cursor", "--cli-failure-report"} {
		hostOnly[name] = true
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			out = append(out, args[i:]...)
			break
		}
		key, _, inline := strings.Cut(token, "=")
		if needsValue, ok := hostOnly[key]; ok {
			if needsValue && !inline && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
			}
			continue
		}
		out = append(out, token)
	}
	return out
}

// ParseAndGetEndpoint get oss endpoint from cli context.
// Priority (aligned with OpenAPI): explicit --endpoint (args/flag) >
// profile.Endpoint > construct from region + endpoint-type.
func ParseAndGetEndpoint(ctx *cli.Context, args []string) (string, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("config failed: %w", err)
	}
	// Ensure flag/env overlays for endpoint fields even when not in configure mode.
	profile.OverwriteWithFlags(ctx)

	return resolveOSSEndpoint(ctx, args, profile)
}

func resolveOSSEndpoint(ctx *cli.Context, args []string, profile config.Profile) (string, error) {
	// 1. Explicit --endpoint in remaining args
	if ep, ok := getArgValue(args, "--endpoint"); ok {
		return ep, nil
	}

	// 2. Explicit --endpoint flag
	if ep, ok := ctx.Flags().GetValue("endpoint"); ok && ep != "" {
		return ep, nil
	}

	// 3. profile.Endpoint (config / env / flag via OverwriteWithFlags)
	if profile.Endpoint != "" {
		return profile.Endpoint, nil
	}

	// 4. Resolve region then build public or VPC/internal endpoint
	region := profile.RegionId
	if r, ok := getArgValue(args, "--region"); ok {
		region = r
	} else if r, ok := ctx.Flags().GetValue("region"); ok && r != "" {
		region = r
	}
	region = strings.TrimSpace(region)
	if region == "" {
		return "", fmt.Errorf("missing region for oss endpoint, use --region <regionId>")
	}
	if !config.IsRegion(region) {
		return "", fmt.Errorf("invalid region %s", region)
	}

	return buildOssEndpoint(region, profile.EndpointType), nil
}

func ParseAndRunCommandFromCli(ctx *cli.Context, args []string) error {
	return parseAndRunCommandFromCli(ctx, args, nil)
}

func parseAndRunCommandFromCli(ctx *cli.Context, args []string, command *Command) (runErr error) {
	defer restoreBridgeFlags(ctx)()
	// Read machine flags before validating the other options, so usage errors
	// also honor an explicit output mode. Skip values and the '--' tail.
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		key, _, inline := strings.Cut(args[i], "=")
		var f *cli.Flag
		if strings.HasPrefix(key, "--") {
			f = ctx.Flags().Get(strings.TrimPrefix(key, "--"))
		} else if len(key) == 2 {
			f = ctx.Flags().GetByShorthand(rune(key[1]))
		}
		if f == nil {
			continue
		}
		end := i + 1
		if f.AssignedMode != cli.AssignedNone && !inline && end < len(args) && !strings.HasPrefix(args[end], "--") {
			end++
		}
		if key == "--cli-output" || key == "--cli-cursor" || key == "--cli-ai-mode" || key == "--no-cli-ai-mode" || key == "--cli-non-interactive" {
			_, _, _ = parseBridgeFlags(ctx, args[i:end])
		}
		i = end - 1
	}
	machine := machineFromContext(ctx)
	previous := activeMachine
	activeMachine = machine
	defer func() {
		if machine.confirmation.Load() && !errors.Is(runErr, errConfirmationRequired) {
			runErr = errors.Join(errConfirmationRequired, runErr)
		}
		runErr = machine.adaptError(runErr)
		activeMachine = previous
	}()
	positional, help, err := parseBridgeFlags(ctx, args)
	if err != nil {
		return err
	}
	machine.ai = machineFromContext(ctx).ai
	machine.preview = machineFromContext(ctx).preview
	for _, name := range []string{"access-key-id", "access-key-secret", "sts-token", "proxy-pwd"} {
		value, _ := ctx.Flags().GetValue(name)
		machine.secrets = append(machine.secrets, value)
	}
	for _, name := range []string{"dryrun", "cli-dry-run", "cli-dry-run-json"} {
		if f := ctx.Flags().Get(name); f != nil && f.IsAssigned() {
			return fmt.Errorf("--%s is not supported by built-in OSS; no operation was executed", name)
		}
	}
	if help {
		return printHostOSSHelp(ctx, nil)
	}
	forwarded := stripCliOnlyFlagsFromArgs(args)
	_, parsed, err := parseOSSOptions(forwarded)
	if err != nil {
		return err
	}
	if err := machine.validate(parsed); err != nil {
		return err
	}
	if command != nil {
		if err := validateCommandArgCount(command, positional); err != nil {
			return err
		}
		validation := *command
		validation.options = parsed
		if err := validation.checkOptions(); err != nil {
			return err
		}
	}
	mode, err := previewMode(ctx)
	if err != nil {
		return err
	}
	if f := ctx.Flags().Get("cli-failure-report"); f != nil && f.IsAssigned() {
		if mode != "" || command == nil || (command.name != "cp" && command.name != "sync") {
			return fmt.Errorf("--cli-failure-report requires cp or sync execution")
		}
		path, _ := f.GetValue()
		if path == "" {
			return fmt.Errorf("--cli-failure-report requires a nonempty path")
		}
	}
	for _, name := range []string{"retry-count", "connect-timeout", "read-timeout"} {
		if value, ok := ctx.Flags().GetValue(name); ok {
			number, err := strconv.ParseInt(value, 10, 64)
			if err != nil || number < 0 {
				return fmt.Errorf("option --%s requires a non-negative integer", name)
			}
		}
	}

	var preview *ossPreview
	if mode != "" {
		preview, err = preparePreview(command, positional, forwarded, parsed, mode)
		if err != nil {
			return err
		}
		if mode == "validate" {
			return preview.run(ctx, *command, parsed, nil)
		}
	}

	if command != nil {
		switch command.name {
		case "update":
			return fmt.Errorf("aliyun oss update is not supported in the host CLI; update aliyun using its installation method")
		case "hash", "help", "config":
			return runLocalOssCommand(ctx, args)
		}
	}

	machine.phase = "configuration"
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return fmt.Errorf("config failed: %w", err)
	}

	profile.OverwriteWithFlags(ctx)

	proxyHost, ok := ctx.Flags().GetValue("proxy-host")
	if !ok {
		proxyHost = ""
	}
	credential, err := profile.GetCredential(ctx, tea.String(proxyHost))
	if err != nil {
		return fmt.Errorf("can't get credential %w", err)
	}

	model, err := credential.GetCredential()
	if err != nil {
		return fmt.Errorf("can't get credential %w", err)
	}

	provider := &hostOSSProvider{source: credential}
	if model != nil {
		if model.AccessKeyId != nil {
			provider.current.id = *model.AccessKeyId
		}
		if model.AccessKeySecret != nil {
			provider.current.secret = *model.AccessKeySecret
		}
		if model.SecurityToken != nil {
			provider.current.token = *model.SecurityToken
		}
	}
	provider.secrets = []string{provider.current.id, provider.current.secret, provider.current.token}
	previousProvider := bridgeCredentialProvider
	bridgeCredentialProvider = provider
	defer func() {
		machine.secrets = append(machine.secrets, provider.secrets...)
		bridgeCredentialProvider = previousProvider
	}()
	// The host has already resolved role/process credentials. The SDK receives
	// that snapshot, not instructions to assume the role a second time.
	configs := map[string]string{"mode": "AK"}
	if model.AccessKeyId != nil {
		configs["access-key-id"] = *model.AccessKeyId
	}

	if model.AccessKeySecret != nil {
		configs["access-key-secret"] = *model.AccessKeySecret
	}

	if model.SecurityToken != nil {
		configs["sts-token"] = *model.SecurityToken
		if *model.SecurityToken != "" {
			configs["mode"] = "StsToken"
		}
	}

	// read endpoint from flags
	endpoint, err := resolveOSSEndpoint(ctx, args, profile)
	if err != nil {
		return fmt.Errorf("parse endpoint failed: %w", err)
	}
	// check use http force
	forceUseHttp := ctx.Insecure()
	if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if forceUseHttp {
			endpoint = "http://" + endpoint
		} else {
			endpoint = "https://" + endpoint
		}
	}
	configs["endpoint"] = endpoint
	// Host timeouts and OSS timeouts are both expressed in seconds. In the
	// current runtime retry-count and OSS retry-times both bound total attempts.
	if profile.RegionId != "" {
		configs["region"] = profile.RegionId
	}
	if profile.ReadTimeout > 0 {
		configs["read-timeout"] = strconv.Itoa(profile.ReadTimeout)
	}
	if profile.ConnectTimeout > 0 {
		configs["connect-timeout"] = strconv.Itoa(profile.ConnectTimeout)
	}
	if profile.RetryCount > 0 {
		configs["retry-times"] = strconv.Itoa(profile.RetryCount)
	}
	if v, err := GetString(OptionRetryTimes, parsed); err == nil && v != "" {
		configs["retry-times"] = v
	}
	if f := ctx.Flags().Get("skip-secure-verify"); f != nil && f.IsAssigned() {
		configs["skip-verify-cert"] = "true"
	}
	resolved := make(OptionMapType)
	for name, opt := range OptionMap {
		if command != nil {
			supported := false
			for _, valid := range command.validOptionNames {
				if name == valid {
					supported = true
					break
				}
			}
			if !supported {
				continue
			}
		}
		if value, ok := configs[strings.TrimPrefix(opt.nameAlias, "--")]; ok && value != "" {
			if opt.optionType == OptionTypeFlagTrue {
				v := true
				resolved[name] = &v
			} else {
				v := value
				resolved[name] = &v
			}
		}
	}
	if err := checkOption(resolved); err != nil {
		return err
	}
	if preview != nil {
		previewCommand := *command
		previewCommand.args = positional
		return preview.run(ctx, previewCommand, parsed, resolved)
	}
	return runBridgeInvocation(ctx, forwarded, resolved)
}

// The legacy executor still uses process globals for filters and command singletons.
// Scope the handoff and restore it even on error; concurrent embedding is unsupported.
var bridgeResolvedOptions OptionMapType

func runBridgeInvocation(ctx *cli.Context, args []string, resolved OptionMapType) error {
	oldArgs, oldOptions := os.Args, bridgeResolvedOptions
	defer func() { os.Args, bridgeResolvedOptions = oldArgs, oldOptions }()
	os.Args = append([]string{"oss", ctx.Command().Name}, args...)
	bridgeResolvedOptions = resolved
	if activeMachine != nil {
		activeMachine.phase = "execution"
		for _, name := range []string{OptionAccessKeyID, OptionAccessKeySecret, OptionSTSToken, OptionProxyPwd} {
			if v, err := GetString(name, resolved); err == nil {
				activeMachine.secrets = append(activeMachine.secrets, v)
			}
		}
	}
	if activeMachine != nil && (activeMachine.structured() || activeMachine.nonInteractive) {
		switch ctx.Command().Name {
		case "cp", "sync", "rm", "restore", "set-acl", "set-meta", "mb":
			stdout := os.Stdout
			os.Stdout = os.Stderr
			defer func() { os.Stdout = stdout }()
		}
	}
	if f := ctx.Flags().Get("cli-failure-report"); f != nil && f.IsAssigned() {
		path, _ := f.GetValue()
		manifest, err := newFailureManifest(path)
		if err != nil {
			return err
		}
		activeMachine.failures = manifest
		runErr := parseAndRunCommandImpl()
		if activeMachine.confirmation.Load() {
			runErr = errors.Join(errConfirmationRequired, runErr)
		}
		return manifest.finish(runErr)
	}
	return parseAndRunCommandImpl()
}

// Parse each occurrence once. Unlike the generic host parser, OSS repeatable
// options consume exactly one token and '--' terminates option parsing.
func parseBridgeFlags(ctx *cli.Context, args []string) ([]string, bool, error) {
	positional := []string{}
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if token == "--help" || token == "-h" {
			return positional, true, nil
		}
		if !strings.HasPrefix(token, "-") || token == "-" {
			positional = append(positional, token)
			continue
		}
		key, value, inline := strings.Cut(token, "=")
		var flag *cli.Flag
		if strings.HasPrefix(key, "--") {
			flag = ctx.Flags().Get(key[2:])
		} else if len(key) == 2 {
			flag = ctx.Flags().GetByShorthand(rune(key[1]))
		}
		if flag == nil {
			return nil, false, fmt.Errorf("unknown OSS option %s", key)
		}
		if flag.AssignedMode == cli.AssignedNone {
			if inline {
				return nil, false, fmt.Errorf("option %s does not take a value", key)
			}
		} else if !inline {
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
				return nil, false, fmt.Errorf("option %s requires a value", key)
			}
			i++
			value = args[i]
		}
		flag.SetAssigned(true)
		flag.SetValue(value)
	}
	return positional, false, nil
}

func ossPositionalArgs(ctx *cli.Context, args []string) []string {
	positional := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		prefix, _, hasInlineValue := cli.SplitStringWithPrefix(arg, "=:")
		var flag *cli.Flag
		switch {
		case strings.HasPrefix(prefix, "--") && len(prefix) > 2:
			flag = ctx.Flags().Get(prefix[2:])
		case strings.HasPrefix(prefix, "-") && len(prefix) == 2:
			flag = ctx.Flags().GetByShorthand(rune(prefix[1]))
		case strings.HasPrefix(prefix, "-"):
			// Let the host parser report unsupported compact flag forms.
			continue
		default:
			positional = append(positional, arg)
			continue
		}
		if flag != nil && flag.AssignedMode != cli.AssignedNone && !hasInlineValue && i+1 < len(args) {
			i++
		}
	}
	return positional
}

func restoreBridgeFlags(ctx *cli.Context) func() {
	type state struct {
		flag     *cli.Flag
		assigned bool
		value    string
		values   []string
	}
	states := []state{}
	for _, flag := range ctx.Flags().Flags() {
		value, _ := flag.GetValue()
		states = append(states, state{flag, flag.IsAssigned(), value, append([]string(nil), flag.GetValues()...)})
	}
	return func() {
		for _, s := range states {
			s.flag.SetAssigned(s.assigned)
			s.flag.SetValue(s.value)
			s.flag.SetValues(s.values)
		}
	}
}

func validateCommandArgCount(command *Command, args []string) error {
	if len(args) < command.minArgc {
		plural := ""
		if command.minArgc > 1 {
			plural = "s"
		}
		return CommandError{command.name, fmt.Sprintf("the command needs at least %d argument%s", command.minArgc, plural)}
	}
	if len(args) > command.maxArgc {
		plural := ""
		if command.maxArgc > 1 {
			plural = "s"
		}
		return CommandError{command.name, fmt.Sprintf("the command needs at most %d argument%s", command.maxArgc, plural)}
	}
	return nil
}

func runLocalOssCommand(ctx *cli.Context, args []string) error {
	return runBridgeInvocation(ctx, stripCliOnlyFlagsFromArgs(args), nil)
}

func stripOptionWithValue(args []string, option string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}
		if arg == option {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, option+"=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}
