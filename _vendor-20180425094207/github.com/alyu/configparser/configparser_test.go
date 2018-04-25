package configparser

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var gConfig *Configuration

const (
	ConfigFilePath         = "/tmp/configparser_test.ini"
	ConfigFilePathSHA     = "b0b06a0a48d952899fe68addd9f699d2043d045f"
	ConfigNewFilePath     = "/tmp/configparser_test_new.ini"
	ConfigNewFilePathSHA = "a1a3f9acbebed990cbbb573d0f2b6fa9aa8ae5f0"

	SectionName1     = "MYSQLD DEFAULT"
	SectionName2     = "MONGODB"
	SectionName3     = "NDB_MGMD DEFAULT"
	SectionNameRegex = "webservers$"

	Key1 = "TotalSendBufferMemory"
	Key2 = "DefaultOperationRedoProblemAction"
	Key3 = "innodb_buffer_pool_size"
	Key4 = "innodb_buffer_pool_instances"

	Key5 = "datadir"
	Key6 = "smallfiles"

	ConfigFileContent = `wsrep_provider_options="gcache.size=128M; evs.keepalive_period=PT3S; evs.inactive_check_period=PT10S; evs.suspect_timeout=PT30S; evs.inactive_timeout=PT1M; evs.consensus_timeout=PT1M; evs.send_window=1024; evs.user_send_window=512;"
# global settings
SendBufferMemory = 20M
ReceiveBufferMemory = 20M

[dc1.webservers]
10.10.10.10
20.20.20.20
dc1.backup.local

[dc2.database]
30.30.30.30
40.40.40.40
dc2.standby.local

[dc2.webservers]
30.30.30.30
40.40.40.40

[TCP DEFAULT]
#SendBufferMemory=20M
#ReceiveBufferMemory=20M

[NDBD DEFAULT]
NoOfReplicas=2
DataDir=/data/mysql/cluster/dev
FileSystemPath=/data/mysql/cluster/dev
#FileSystemPathDD=
#FileSystemPathDataFiles=
#FileSystemPathUndoFiles=
#BackupDataDir=
#InitialLogFileGroup=name=lg1;undo_buffer_size:64M;undo1.log:64M
#InitialTablespace=name=ts1;extent_size:1M;data1.dat:256M;data2.dat:256M

DataMemory:256M
IndexMemory:32M
DiskPageBufferMemory:64M
SharedGlobalMemory=128M
RedoBuffer=48M
TotalSendBufferMemory=20M

LockPagesInMainMemory=1
Numa=0

RealtimeScheduler=1
MaxNoOfExecutionThreads=4
#LockExecuteThreadToCPU=
#LockMaintThreadsToCPU=
DiskIOThreadPool=2

BuildIndexThreads=2
TwoPassInitialNodeRestartCopy=1

DiskCheckpointSpeedInRestart=100M
DiskCheckpointSpeed=10M

FragmentLogFileSize=256M
NoOfFragmentLogFiles=6
InitFragmentLogFiles=SPARSE

ODirect=1
;CompressedBackup=0
;CompressedLCP=0
Diskless=0

TimeBetweenLocalCheckpoints=20
TimeBetweenGlobalCheckpoints=2000
TimeBetweenEpochs=100
;This parameter defines a timeout for synchronization epochs for MySQL Cluster Replication. If a node fails to participate in a global checkpoint within the time determined by this parameter, the node is shut down
#TimeBetweenEpochsTimeout=4000
# Set in production
#TimeBetweenInactiveTransactionAbortCheck=1000
#TransactionDeadlockDetectionTimeout=1200
#TransactionInactiveTimeout=0
# Might need to increase initial check for large data memory allocations
#TimeBetweenWatchDogCheckInitial = 6000
#TimeBetweenWatchDogCheck= 6000

MaxNoOfConcurrentOperations=250000
MaxNoOfConcurrentScans=500
#MaxNoOfLocalScans=2048

#MaxNoOfConcurrentScans=256 (2-500)
#MaxNoOfLocalScans=numOfDataNodes*MaxNoOfConcurrentScans
# 1-992
#BatchSizePerLocalScan=900
#MaxParallelScansPerFragment=256 (1-1G)

# % of max value
StringMemory=25
MaxNoOfTables=2048
MaxNoOfOrderedIndexes=1024
MaxNoOfUniqueHashIndexes=1024
MaxNoOfAttributes=8192
MaxNoOfTriggers=8192

#MemReportFrequency=10
StartupStatusReportFrequency=10

### Params for setting logging
LogLevelStartup=15
LogLevelShutdown=15
LogLevelCheckpoint=8
LogLevelNodeRestart=15
LogLevelCongestion=15
LogLevelStatistic=15

### Params for increasing Disk throughput
BackupDataBufferSize=16M
BackupLogBufferSize=16M
BackupMemory=32M
#If BackupDataBufferSize and BackupLogBufferSize taken together exceed the default value for BackupMemory, then this parameter must be set explicitly in the config.ini file to their sum.
BackupWriteSize=256K
BackupMaxWriteSize=1M
BackupReportFrequency=10

### CGE 6.3 - REALTIME EXTENSIONS
#RealTimeScheduler=1
#SchedulerExecutionTimer=80
#SchedulerSpinTimer=40

RedoOverCommitCounter=3
RedoOverCommitLimit=20

StartFailRetryDelay=0
MaxStartFailRetries=3

[NDB_MGMD DEFAULT]
PortNumber=1186
DataDir=/data/mysql/cluster/dev
#MaxNoOfSavedEvents=100
TotalSendBufferMemory=4M

[NDB_MGMD]
NodeId=1
HostName=localhost
PortNumber=1186
ArbitrationRank=1

#[NDB_MGMD]
#NodeId=2
#HostName=localhost
#PortNumber=1187
#ArbitrationRank=1

[NDBD]
NodeId=10
HostName=localhost
#HeartbeatOrder=10

[NDBD]
NodeId=11
HostName=localhost
#HeartbeatOrder=20

[NDBD]
NodeId=12
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=13
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=14
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

[NDBD]
NodeId=15
HostName=localhost
#HeartbeatOrder=20
NodeGroup=65536

#
# Note=The following can be MySQLD connections or
#      NDB API application connecting to the cluster
#
[MYSQLD DEFAULT]
TotalSendBufferMemory=10M
DefaultOperationRedoProblemAction=ABORT
#DefaultOperationRedoProblemAction=QUEUE
#BatchByteSize=32K (1024-1M)
# 1-992
#BatchSize=900
#MaxScanBatchSize=256K (32K-16M)
; this is another comment
[MYSQLD]
NodeId=100
HostName=localhost

[API]
NodeId=101
[API]
NodeId=102
[API]
NodeId=103
[API]
NodeId=104
[API]
NodeId=105
[API]
NodeId=106
[API]
NodeId=107
[API]
NodeId=108
[API]
NodeId=109
[API]
NodeId=110
[API]
NodeId=111

[API]
NodeId=200

[API]
NodeId=201
[API]
NodeId=202
[API]
NodeId=203
[API]
NodeId=204
[API]
NodeId=205
[API]
NodeId=206
[API]
NodeId=207
[API]
NodeId=208
[API]
NodeId=209
[API]
NodeId=210
[API]
NodeId=211

[API]
NodeId=212
[API]
NodeId=213
[API]
NodeId=214
[API]
NodeId=215
[API]
NodeId=216
[API]
NodeId=217
[API]
NodeId=218
[API]
NodeId=219
[API]
NodeId=220
[API]
NodeId=221
[API]
NodeId=222
`
)

func TestWriteTestConfigFile(t *testing.T) {
	t.Log("Writing test config to" + ConfigFilePath)
	f, err := os.Create(ConfigFilePath)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = f.Close()
	}()

	w := bufio.NewWriter(f)
	defer func() {
		err = w.Flush()
	}()

	w.WriteString(ConfigFileContent)
}

func TestReadConfigFile(t *testing.T) {
	t.Log("Reading test config " + ConfigFilePath)

	var err error
	gConfig, err = Read(ConfigFilePath)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(gConfig)
}

func TestGetSection(t *testing.T) {
	s, err := getConfig().Section("NDBD DEFAULT")
	if err != nil {
		t.Error(err)
	}

	t.Log(s)
}

func TestGetSections(t *testing.T) {
	s, err := getConfig().Sections("NDBD")
	if err != nil {
		t.Error(err)
	}

	t.Log(s)
}

func TestSetNewValue(t *testing.T) {
	s, err := getConfig().Section(SectionName1)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%s=%s\n", Key1, s.ValueOf(Key1))
	oldValue := s.SetValueFor(Key1, "512M")
	t.Logf("New: %s=%s\n", Key1, s.ValueOf(Key1))
	if oldValue == s.ValueOf(Key1) {
		t.Error("Unable to change value for key " + s.ValueOf(Key1))
	}
}

func TestAddOption(t *testing.T) {
	s, err := getConfig().Section(SectionName1)
	if err != nil {
		t.Error(err)
	}

	testAddOption(s, Key3, "128G", t)
	testAddOption(s, Key4, "16", t)

	testAddOption(s, Key3, "64G", t)
	testAddOption(s, Key4, "8", t)
}

func TestDeleteOption(t *testing.T) {
	s, err := getConfig().Section(SectionName1)
	if err != nil {
		t.Error(err)
	}

	testDeleteOption(s, Key2, t)
}

func TestNotExistsOption(t *testing.T) {
	s, err := getConfig().Section(SectionName1)
	if err != nil {
		t.Error(err)
	}

	if s.Exists("none_existing_key") {
		t.Error("none existing key found")
	}
}

func TestNewSection(t *testing.T) {
	s := getConfig().NewSection(SectionName2)
	s.Add(Key5, "/var/lib/mongodb")
	s.Add(Key6, "true")
}

func TestGetNewSections(t *testing.T) {
	s, err := getConfig().Section(SectionName2)
	if err != nil {
		t.Error(err)
	}
	if !s.Exists(Key5) {
		t.Error(Key5 + " does not exists")
	}

	if !s.Exists(Key6) {
		t.Error(Key6 + " does not exists")
	}

	t.Log(s)
}

func TestDeleteSection(t *testing.T) {
	c := getConfig()
	sections, err := c.Delete(SectionName3)
	if err != nil {
		t.Error(err)
	}
	for _, s := range sections {
		t.Log(s)
	}
}

func TestFindSection(t *testing.T) {
	c := getConfig()
	sections, err := c.Find(SectionNameRegex)
	if err != nil {
		t.Error(err)
	}
	for _, s := range sections {
		t.Log(s)
	}
}

func TestSaveNewConfigFile(t *testing.T) {
	c := getConfig()

	err := Save(c, ConfigNewFilePath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSectionString(t *testing.T) {
	Delimiter = " = "
	s, err := getConfig().Section(SectionName2)
	if err != nil {
		t.Error(err)
	}

	expected := `[MONGODB]
datadir = /var/lib/mongodb
smallfiles = true
`
	str := s.String()
	t.Log(expected)
	t.Log(str)
	if str != expected {
		t.Error("section string doesn't match expected string")
	}
}

func TestSHA(t *testing.T) {
	out, err := exec.Command("shasum", ConfigNewFilePath).Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Split(string(out), " ")
	t.Logf("%v=%v", sha[0], ConfigNewFilePathSHA)
	if sha[0] != ConfigNewFilePathSHA {
		t.Error(ConfigNewFilePath + " shasum doees not match!")
	}
}

func getConfig() *Configuration {
	if gConfig == nil {
		log.Println("No configuration instance!")
		os.Exit(1)
	}
	return gConfig
}

func testAddOption(s *Section, name string, value string, t *testing.T) {
	oldValue := s.Add(name, value)
	t.Logf("%s=%s, old value: %s\n", name, s.ValueOf(name), oldValue)
	if oldValue == s.ValueOf(name) {
		t.Error("Unable to change value for key " + s.ValueOf(name))
	}
}

func testDeleteOption(s *Section, name string, t *testing.T) {
	oldValue := s.Delete(name)
	t.Logf("%s=%s\n", name, oldValue)
}
