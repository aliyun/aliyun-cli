package lib

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var commandLine string

func LogEnd(startT time.Time) {
	LogInfo("ossutil run end,cost:%d(ms).\n", time.Now().UnixNano()/1000/1000-startT.UnixNano()/1000/1000)
	UnInitLogger()
}

// ParseAndRunCommand parse command line user input, get command and options, then run command
func ParseAndRunCommand() error {
	ts := time.Now().UnixNano()

	commandLine = getCommandLine()

	clearEnv()

	args, options, err := ParseArgOptions()
	if err != nil {
		return err
	}

	var level = oss.LogOff
	strLevel, err := getLoglevelFromOptions(options)
	if strLevel == "info" {
		level = oss.Info
	} else if strLevel == "debug" {
		level = oss.Debug
	} else if len(strLevel) > 0 {
		return fmt.Errorf("loglevel must be:info|debug")
	}

	if level > oss.LogOff {
		InitLogger(level, logName)
	}

	startT := time.Now()
	LogInfo("ossutil run begin,cmd:%s\n", commandLine)
	LogInfo("ossutil version is %s\n", Version)
	LogInfo("oss go sdk version is %s\n", oss.Version)
	LogInfo("go version is %s\n", runtime.Version())
	LogInfo("runtime.NumCPU is %d\n", runtime.NumCPU())

	defer LogEnd(startT)

	showElapse, err := RunCommand(args, options)
	if err != nil {
		LogError("%s.\n", err.Error())
		return err
	}
	if showElapse {
		te := time.Now().UnixNano()
		fmt.Printf("\n%.6f(s) elapsed\n", float64(te-ts)/1e9)
		return nil
	}
	return nil
}

func getCommandLine() string {
	return strings.Join(os.Args, " ")
}

func clearEnv() {
	if runtime.GOOS == "windows" {
		_, renameFilePath := getBinaryPath()
		os.Remove(renameFilePath)
	}
}

func RunCommand(args []string, options OptionMapType) (bool, error) {
	if len(args) == 0 {
		if val, _ := GetBool(OptionVersion, options); val {
			fmt.Printf("ossutil version: %s\n", Version)
			return false, nil
		}
		args = append(args, "help")
	}
	command := args[0]
	args = args[1:]

	cm := CommandManager{}
	cm.Init()
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

// CommandManager is used to manager commands, such as build command map and run command
type CommandManager struct {
	commandMap map[string]interface{}
}

// Init build command map
func (cm *CommandManager) Init() {
	commandList := GetAllCommands()
	cm.commandMap = make(map[string]interface{}, len(commandList))

	for _, cmd := range commandList {
		name := reflect.ValueOf(cmd).Elem().FieldByName("command").FieldByName("name").String()
		cm.commandMap[name] = cmd
	}
}

// RunCommand select command from command map, initialize command and run command
func (cm *CommandManager) RunCommand(commandName string, args []string, options OptionMapType) (bool, error) {

	if cmd, ok := cm.commandMap[commandName]; ok {
		if err := cmd.(Commander).Init(args, options); err != nil {
			return false, err
		}
		if err := cmd.(Commander).RunCommand(); err != nil {
			return false, err
		}
		group := reflect.ValueOf(cmd).Elem().FieldByName("command").FieldByName("group").String()
		return group == GroupTypeNormalCommand, nil
	}
	return false, fmt.Errorf("no such command: \"%s\", please try \"help\" for more information", commandName)
}

func getLoglevelFromOptions(options OptionMapType) (string, error) {
	strLevel, err := GetString(OptionLogLevel, options)
	if err != nil {
		return "", err
	}
	if strLevel != "" {
		return strLevel, nil
	}

	configFile, _ := GetString(OptionConfigFile, options)

	strLevel, err = readLoglevelFromFile(configFile)
	if err != nil {
		return "", err
	}

	return strLevel, nil
}
