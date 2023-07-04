package lib

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	normalExit = iota
	errExit
)

var processTickInterval int64 = 5

var clearStrLen int = 0
var clearStr string = strings.Repeat(" ", clearStrLen)

func getClearStr(str string) string {
	if clearStrLen <= len(str) {
		clearStrLen = len(str)
		return fmt.Sprintf("\r%s", str)
	}
	clearStr = strings.Repeat(" ", clearStrLen)
	return fmt.Sprintf("\r%s\r%s", clearStr, str)
}

type Monitorer interface {
	setScanError(err error)
	updateScanNum(num int64)
	setScanEnd()
}

// for normal object operation
type MonitorSnap struct {
	okNum   int64
	errNum  int64
	skipNum int64
	dealNum int64
}

/*
 * Put same type variables together to make them 64bits alignment to avoid
 * atomic.AddInt64() panic
 * Please guarantee the alignment if you add new filed
 */
type Monitor struct {
	opStr          string
	totalNum       int64
	okNum          int64
	errNum         int64
	skipNum        int64
	seekAheadError error
	seekAheadEnd   bool
	finish         bool
	_              uint32 //Add padding to make sure the next data 64bits alignment
}

func (m *Monitor) init(opStr string) {
	m.opStr = opStr
	m.totalNum = 0
	m.seekAheadEnd = false
	m.seekAheadError = nil
	m.okNum = 0
	m.errNum = 0
	m.skipNum = 0
	m.finish = false
}

func (m *Monitor) setScanError(err error) {
	m.seekAheadError = err
	m.seekAheadEnd = true
}

func (m *Monitor) updateScanNum(num int64) {
	m.totalNum = m.totalNum + num
}

func (m *Monitor) setScanEnd() {
	m.seekAheadEnd = true
}

func (m *Monitor) updateOKNum(num int64) {
	atomic.AddInt64(&m.okNum, num)
}

func (m *Monitor) updateErrNum(num int64) {
	atomic.AddInt64(&m.errNum, num)
}

func (m *Monitor) getSnapshot() *MonitorSnap {
	var snap MonitorSnap
	snap.okNum = m.okNum
	snap.errNum = m.errNum
	snap.skipNum = m.skipNum
	snap.dealNum = snap.okNum + snap.errNum
	return &snap
}

func (m *Monitor) progressBar(finish bool, exitStat int) string {
	if m.finish {
		return ""
	}
	m.finish = m.finish || finish
	if !finish {
		return m.getProgressBar()
	}
	return m.getFinishBar(exitStat)
}

func (m *Monitor) getProgressBar() string {
	snap := m.getSnapshot()
	if m.seekAheadEnd && m.seekAheadError == nil {
		if snap.errNum == 0 {
			return getClearStr(fmt.Sprintf("Total %d objects. %s %d objects, Progress: %d%s", m.totalNum, m.opStr, snap.okNum, m.getPrecent(snap), "%%"))
		}
		return getClearStr(fmt.Sprintf("Total %d objects. %s %d objects, Error %d objects, Progress: %d%s", m.totalNum, m.opStr, snap.okNum, snap.errNum, m.getPrecent(snap), "%%"))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return getClearStr(fmt.Sprintf("Scanned %d objects. %s %d objects.", scanNum, m.opStr, snap.okNum))
	}
	return getClearStr(fmt.Sprintf("Scanned %d objects. %s %d objects, Error %d objects.", scanNum, m.opStr, snap.okNum, snap.errNum))
}

func (m *Monitor) getPrecent(snap *MonitorSnap) int {
	if m.seekAheadEnd && m.seekAheadError == nil {
		if m.totalNum != 0 {
			return int(float64((snap.dealNum)*100.0) / float64(m.totalNum))
		}
		return 100
	}
	return 0
}

func (m *Monitor) getFinishBar(exitStat int) string {
	if exitStat == normalExit {
		return m.getWholeFinishBar()
	}
	return m.getDefeatBar()
}

func (m *Monitor) getWholeFinishBar() string {
	snap := m.getSnapshot()
	if m.seekAheadEnd && m.seekAheadError == nil {
		if snap.errNum == 0 {
			return getClearStr(fmt.Sprintf("Succeed: Total %d objects. %s %d objects(skip %d objects).\n", m.totalNum, m.opStr, snap.okNum, snap.skipNum))
		}
		return getClearStr(fmt.Sprintf("FinishWithError: Total %d objects. %s %d objects(skip %d objects), Error %d objects.\n", m.totalNum, m.opStr, snap.okNum, snap.skipNum, snap.errNum))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return getClearStr(fmt.Sprintf("Succeed: Total %d objects. %s %d objects(skip %d objects).\n", scanNum, m.opStr, snap.okNum, snap.skipNum))
	}
	return getClearStr(fmt.Sprintf("FinishWithError: Scanned %d objects. %s %d objects(skip %d objects), Error %d objects.\n", scanNum, m.opStr, snap.okNum, snap.skipNum, snap.errNum))
}

func (m *Monitor) getDefeatBar() string {
	snap := m.getSnapshot()
	if m.seekAheadEnd && m.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total %d objects. %s %d objects(skip %d objects), when error happens.\n", m.totalNum, m.opStr, snap.okNum, snap.skipNum))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	return getClearStr(fmt.Sprintf("Scanned %d objects. %s %d objects(skip %d objects), when error happens.\n", scanNum, m.opStr, snap.okNum, snap.skipNum))
}

// For rm
type RMMonitorSnap struct {
	objectNum      int64
	uploadIdNum    int64
	errObjectNum   int64
	errUploadIdNum int64
	dealNum        int64
	errNum         int64
	removedBucket  string
}

/*
 * Put same type variables together to make them 64bits alignment to avoid
 * atomic.AddInt64() panic
 * Please guarantee the alignment if you add new filed
 */
type RMMonitor struct {
	op               int64
	totalObjectNum   int64
	totalUploadIdNum int64
	objectNum        int64
	uploadIdNum      int64
	errObjectNum     int64
	errUploadIdNum   int64
	removedBucket    string
	seekAheadError   error
	seekAheadEnd     bool
	finish           bool
	_                uint32 //Add padding to make sure the next data 64bits alignment
}

func (m *RMMonitor) init() {
	m.op = 0
	m.totalObjectNum = 0
	m.totalUploadIdNum = 0
	m.seekAheadEnd = false
	m.seekAheadError = nil
	m.objectNum = 0
	m.uploadIdNum = 0
	m.errObjectNum = 0
	m.errUploadIdNum = 0
	m.finish = false
	m.removedBucket = ""
}

func (m *RMMonitor) updateOP(op int64) {
	m.op = m.op | op
}

func (m *RMMonitor) setOP(op int64) {
	m.op = op
}

func (m *RMMonitor) setScanError(err error) {
	m.seekAheadError = err
	m.seekAheadEnd = true
}

func (m *RMMonitor) updateScanNum(num int64) {
	m.totalObjectNum = m.totalObjectNum + num
}

func (m *RMMonitor) updateScanUploadIdNum(num int64) {
	m.totalUploadIdNum = m.totalUploadIdNum + num
}

func (m *RMMonitor) setScanEnd() {
	m.seekAheadEnd = true
}

func (m *RMMonitor) updateObjectNum(num int64) {
	atomic.AddInt64(&m.objectNum, num)
}

func (m *RMMonitor) updateUploadIdNum(num int64) {
	atomic.AddInt64(&m.uploadIdNum, num)
}

func (m *RMMonitor) updateErrObjectNum(num int64) {
	atomic.AddInt64(&m.errObjectNum, num)
}

func (m *RMMonitor) updateErrUploadIdNum(num int64) {
	atomic.AddInt64(&m.errUploadIdNum, num)
}

func (m *RMMonitor) updateRemovedBucket(bucket string) {
	m.removedBucket = bucket
}

func (m *RMMonitor) getSnapshot() *RMMonitorSnap {
	var snap RMMonitorSnap
	snap.objectNum = m.objectNum
	snap.uploadIdNum = m.uploadIdNum
	snap.errObjectNum = m.errObjectNum
	snap.errUploadIdNum = m.errUploadIdNum
	snap.dealNum = snap.objectNum + snap.uploadIdNum + snap.errObjectNum + snap.errUploadIdNum
	snap.errNum = snap.errObjectNum + snap.errUploadIdNum
	snap.removedBucket = m.removedBucket
	return &snap
}

func (m *RMMonitor) progressBar(finish bool, exitStat int) string {
	if m.finish {
		return ""
	}
	m.finish = m.finish || finish
	if !finish {
		return m.getProgressBar()
	}
	return m.getFinishBar(exitStat)
}

func (m *RMMonitor) getProgressBar() string {
	if m.op&allType != 0 {
		snap := m.getSnapshot()
		if m.seekAheadEnd && m.seekAheadError == nil {
			return getClearStr(fmt.Sprintf("Total %s. %s%s Progress: %d%s", m.getTotalInfo(), m.getOKInfo(snap), m.getErrInfo(snap), m.getPrecent(snap), "%%"))
		}
		m.totalObjectNum = max(m.totalObjectNum, snap.objectNum+snap.errObjectNum)
		m.totalUploadIdNum = max(m.totalUploadIdNum, snap.uploadIdNum+snap.errUploadIdNum)
		return getClearStr(fmt.Sprintf("Scanned %s. %s%s", m.getTotalInfo(), m.getOKInfo(snap), m.getErrInfo(snap)))
	}
	return getClearStr("")
}

func (m *RMMonitor) getTotalInfo() string {
	strList := []string{}
	if m.op&objectType != 0 {
		strList = append(strList, fmt.Sprintf("%d objects", m.totalObjectNum))
	}
	if m.op&multipartType != 0 {
		strList = append(strList, fmt.Sprintf("%d uploadIds", m.totalUploadIdNum))
	}
	return strings.Join(strList, ", ")
}

func (m *RMMonitor) getOKInfo(snap *RMMonitorSnap) string {
	strList := []string{}
	if m.op&allType == 0 {
		return ""
	}
	if m.op&objectType != 0 {
		strList = append(strList, fmt.Sprintf("%d objects", snap.objectNum))
	}
	if m.op&multipartType != 0 {
		strList = append(strList, fmt.Sprintf("%d uploadIds", snap.uploadIdNum))
	}
	return fmt.Sprintf("Removed %s.", strings.Join(strList, ", "))
}

func (m *RMMonitor) getErrInfo(snap *RMMonitorSnap) string {
	if snap.errNum != 0 {
		strList := []string{}
		if snap.errObjectNum != 0 {
			strList = append(strList, fmt.Sprintf("%d objects", snap.errObjectNum))
		}
		if snap.errUploadIdNum != 0 {
			strList = append(strList, fmt.Sprintf("%d uploadIds", snap.errUploadIdNum))
		}
		return fmt.Sprintf(" Error %s.", strings.Join(strList, ", "))
	}
	return ""
}

func (m *RMMonitor) getPrecent(snap *RMMonitorSnap) int {
	if m.seekAheadEnd && m.seekAheadError == nil {
		if m.totalObjectNum+m.totalUploadIdNum != 0 {
			return int(float64((snap.dealNum)*100.0) / float64(m.totalObjectNum+m.totalUploadIdNum))
		}
		return 100
	}
	return 0
}

func (m *RMMonitor) getFinishBar(exitStat int) string {
	snap := m.getSnapshot()
	return m.getObjectFinishBar(snap, exitStat) + m.getBucketFinishBar(snap)
}

func (m *RMMonitor) getObjectFinishBar(snap *RMMonitorSnap, exitStat int) string {
	if m.op&allType != 0 {
		if m.seekAheadEnd && m.seekAheadError == nil {
			if m.getExitStat(snap, exitStat) == errExit {
				return getClearStr(fmt.Sprintf("Total %s. %s when error happens.\n", m.getTotalInfo(), m.getOKInfo(snap)))
			}
			return getClearStr(fmt.Sprintf("Succeed: Total %s. %s\n", m.getTotalInfo(), m.getOKInfo(snap)))
		}
		m.totalObjectNum = max(m.totalObjectNum, snap.objectNum+snap.errObjectNum)
		m.totalUploadIdNum = max(m.totalUploadIdNum, snap.uploadIdNum+snap.errUploadIdNum)
		if m.getExitStat(snap, exitStat) == errExit {
			return getClearStr(fmt.Sprintf("Scanned %s. %s when error happens.\n", m.getTotalInfo(), m.getOKInfo(snap)))
		}
		return getClearStr(fmt.Sprintf("Succeed: Total %s. %s\n", m.getTotalInfo(), m.getOKInfo(snap)))
	}
	return getClearStr("")
}

func (m *RMMonitor) getExitStat(snap *RMMonitorSnap, exitStat int) int {
	if exitStat != normalExit || snap.errNum != 0 || (m.op&bucketType != 0 && snap.removedBucket == "") {
		return errExit
	}
	return normalExit
}

func (m *RMMonitor) getBucketFinishBar(snap *RMMonitorSnap) string {
	if m.op&bucketType != 0 && snap.removedBucket != "" {
		return getClearStr(fmt.Sprintf("Removed Bucket: %s\n", snap.removedBucket))
	}
	return getClearStr("")
}

// for cp
type CPMonitorSnap struct {
	transferSize  int64
	skipSize      int64
	dealSize      int64
	fileNum       int64
	dirNum        int64
	skipNum       int64
	skipNumDir    int64
	errNum        int64
	okNum         int64
	dealNum       int64
	duration      int64
	incrementSize int64
}

/*
 * Put same type variables together to make them 64bits alignment to avoid
 * atomic.AddInt64() panic
 * Please guarantee the alignment if you add new filed
 */
type CPMonitor struct {
	totalSize      int64
	totalNum       int64
	transferSize   int64
	skipSize       int64
	dealSize       int64
	fileNum        int64
	dirNum         int64
	skipNum        int64
	skipNumDir     int64
	errNum         int64
	lastSnapSize   int64
	tickDuration   int64
	seekAheadError error
	op             operationType
	seekAheadEnd   bool
	finish         bool
	_              uint32 //Add padding to make sure the next data 64bits alignment
	lastSnapTime   time.Time
}

func (m *CPMonitor) init(op operationType) {
	m.op = op
	m.totalSize = 0
	m.totalNum = 0
	m.seekAheadEnd = false
	m.seekAheadError = nil
	m.transferSize = 0
	m.skipSize = 0
	m.dealSize = 0
	m.fileNum = 0
	m.dirNum = 0
	m.skipNum = 0
	m.errNum = 0
	m.finish = false
	m.lastSnapSize = 0
	m.lastSnapTime = time.Now()
	m.tickDuration = processTickInterval * int64(time.Second)
}

func (m *CPMonitor) setScanError(err error) {
	m.seekAheadError = err
	m.seekAheadEnd = true
}

func (m *CPMonitor) updateScanNum(num int64) {
	m.totalNum = m.totalNum + num
}

func (m *CPMonitor) updateScanSizeNum(size, num int64) {
	m.totalSize = m.totalSize + size
	m.totalNum = m.totalNum + num
}

func (m *CPMonitor) setScanEnd() {
	m.seekAheadEnd = true
}

func (m *CPMonitor) updateTransferSize(size int64) {
	atomic.AddInt64(&m.transferSize, size)
}

func (m *CPMonitor) updateDealSize(size int64) {
	atomic.AddInt64(&m.dealSize, size)
}

func (m *CPMonitor) updateFile(size, num int64) {
	atomic.AddInt64(&m.fileNum, num)
	atomic.AddInt64(&m.transferSize, size)
	atomic.AddInt64(&m.dealSize, size)
}

func (m *CPMonitor) updateDir(size, num int64) {
	atomic.AddInt64(&m.dirNum, num)
	atomic.AddInt64(&m.transferSize, size)
	atomic.AddInt64(&m.dealSize, size)
}

func (m *CPMonitor) updateSkip(size, num int64) {
	atomic.AddInt64(&m.skipNum, num)
	atomic.AddInt64(&m.skipSize, size)
}

func (m *CPMonitor) updateSkipDir(num int64) {
	atomic.AddInt64(&m.skipNumDir, num)
}

func (m *CPMonitor) updateErr(size, num int64) {
	atomic.AddInt64(&m.errNum, num)
	atomic.AddInt64(&m.transferSize, size)
}

func (m *CPMonitor) getSnapshot() *CPMonitorSnap {
	var snap CPMonitorSnap
	snap.transferSize = m.transferSize
	snap.skipSize = m.skipSize
	snap.dealSize = m.dealSize + snap.skipSize
	snap.fileNum = m.fileNum
	snap.dirNum = m.dirNum
	snap.skipNum = m.skipNum
	snap.errNum = m.errNum
	snap.okNum = snap.fileNum + snap.dirNum + snap.skipNum
	snap.dealNum = snap.okNum + snap.errNum
	snap.skipNumDir = m.skipNumDir
	now := time.Now()
	snap.duration = now.Sub(m.lastSnapTime).Nanoseconds()

	return &snap
}

func (m *CPMonitor) progressBar(finish bool, exitStat int) string {
	if m.finish {
		return ""
	}
	m.finish = m.finish || finish
	if !finish {
		return m.getProgressBar()
	}
	return m.getFinishBar(exitStat)
}

func (m *CPMonitor) getProgressBar() string {
	mu.RLock()
	defer mu.RUnlock()

	snap := m.getSnapshot()
	if snap.duration < m.tickDuration {
		return ""
	} else {
		m.lastSnapTime = time.Now()
		snap.incrementSize = m.transferSize - m.lastSnapSize
		m.lastSnapSize = snap.transferSize
	}

	if m.seekAheadEnd && m.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Dealed num: %d%s%s, Progress: %.3f%s, Speed: %.2fKB/s", m.totalNum, getSizeString(m.totalSize), snap.dealNum, m.getDealNumDetail(snap), m.getDealSizeDetail(snap), m.getPrecent(snap), "%%", m.getSpeed(snap)))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	scanSize := max(m.totalSize, snap.dealSize)
	return getClearStr(fmt.Sprintf("Scanned num: %d, size: %s. Dealed num: %d%s%s, Speed: %.2fKB/s.", scanNum, getSizeString(scanSize), snap.dealNum, m.getDealNumDetail(snap), m.getDealSizeDetail(snap), m.getSpeed(snap)))
}

func (m *CPMonitor) getFinishBar(exitStat int) string {
	if exitStat == normalExit {
		return m.getWholeFinishBar()
	}
	return m.getDefeatBar()
}

func (m *CPMonitor) getWholeFinishBar() string {
	snap := m.getSnapshot()
	if m.seekAheadEnd && m.seekAheadError == nil {
		if snap.errNum == 0 {
			return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", m.totalNum, getSizeString(m.totalSize), snap.okNum, m.getDealNumDetail(snap), m.getSkipSize(snap)))
		}
		return getClearStr(fmt.Sprintf("FinishWithError: Total num: %d, size: %s. Error num: %d. OK num: %d%s%s.\n", m.totalNum, getSizeString(m.totalSize), snap.errNum, snap.okNum, m.getOKNumDetail(snap), m.getSizeDetail(snap)))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	if snap.errNum == 0 {
		return getClearStr(fmt.Sprintf("Succeed: Total num: %d, size: %s. OK num: %d%s%s.\n", scanNum, getSizeString(snap.dealSize), snap.okNum, m.getDealNumDetail(snap), m.getSkipSize(snap)))
	}
	return getClearStr(fmt.Sprintf("FinishWithError: Scanned %d %s. Error num: %d. OK num: %d%s%s.\n", scanNum, m.getSubject(), snap.errNum, snap.okNum, m.getOKNumDetail(snap), m.getSizeDetail(snap)))
}

func (m *CPMonitor) getDefeatBar() string {
	snap := m.getSnapshot()
	if m.seekAheadEnd && m.seekAheadError == nil {
		return getClearStr(fmt.Sprintf("Total num: %d, size: %s. Dealed num: %d%s%s. When error happens.\n", m.totalNum, getSizeString(m.totalSize), snap.okNum, m.getOKNumDetail(snap), m.getSizeDetail(snap)))
	}
	scanNum := max(m.totalNum, snap.dealNum)
	return getClearStr(fmt.Sprintf("Scanned %d %s. Dealed num: %d%s%s. When error happens.\n", scanNum, m.getSubject(), snap.okNum, m.getOKNumDetail(snap), m.getSizeDetail(snap)))
}

func (m *CPMonitor) getSubject() string {
	switch m.op {
	case operationTypePut:
		return "files"
	default:
		return "objects"
	}
}

func (m *CPMonitor) getDealNumDetail(snap *CPMonitorSnap) string {
	return m.getNumDetail(snap, true)
}

func (m *CPMonitor) getOKNumDetail(snap *CPMonitorSnap) string {
	return m.getNumDetail(snap, false)
}

func (m *CPMonitor) getNumDetail(snap *CPMonitorSnap, hasErr bool) string {
	if !hasErr && snap.okNum == 0 {
		return ""
	}
	strList := []string{}
	if hasErr && snap.errNum != 0 {
		strList = append(strList, fmt.Sprintf("Error %d %s", snap.errNum, m.getSubject()))
	}
	if snap.fileNum != 0 {
		strList = append(strList, fmt.Sprintf("%s %d %s", m.getOPStr(), snap.fileNum, m.getSubject()))
	}
	if snap.dirNum != 0 {
		str := fmt.Sprintf("%d directories", snap.dirNum)
		if snap.fileNum == 0 {
			str = fmt.Sprintf("%s %d directories", m.getOPStr(), snap.dirNum)
		}
		strList = append(strList, str)
	}
	if snap.skipNum != 0 {
		strList = append(strList, fmt.Sprintf("skip %d %s", snap.skipNum, m.getSubject()))
	}
	if snap.skipNumDir != 0 {
		strList = append(strList, fmt.Sprintf("skip %d directory", snap.skipNumDir))
	}

	if len(strList) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(strList, ", "))
}

func (m *CPMonitor) getSpeed(snap *CPMonitorSnap) float64 {
	return (float64(snap.incrementSize) / 1024) / (float64(snap.duration) * 1e-9)
}

func (m *CPMonitor) getOPStr() string {
	switch m.op {
	case operationTypePut:
		return "upload"
	case operationTypeGet:
		return "download"
	default:
		return "copy"
	}
}

func (m *CPMonitor) getDealSizeDetail(snap *CPMonitorSnap) string {
	return fmt.Sprintf(", OK size: %s", getSizeString(snap.dealSize))
}

func (m *CPMonitor) getSkipSize(snap *CPMonitorSnap) string {
	if snap.skipSize != 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return ""
}

func (m *CPMonitor) getSizeDetail(snap *CPMonitorSnap) string {
	if snap.skipSize == 0 {
		return fmt.Sprintf(", Transfer size: %s", getSizeString(snap.transferSize))
	}
	if snap.transferSize == 0 {
		return fmt.Sprintf(", Skip size: %s", getSizeString(snap.skipSize))
	}
	return fmt.Sprintf(", OK size: %s(transfer: %s, skip: %s)", getSizeString(snap.transferSize+snap.skipSize), getSizeString(snap.transferSize), getSizeString(snap.skipSize))
}

func (m *CPMonitor) getPrecent(snap *CPMonitorSnap) float64 {
	if m.seekAheadEnd && m.seekAheadError == nil {
		if m.totalSize != 0 {
			return float64((snap.dealSize)*100.0) / float64(m.totalSize)
		}
		if m.totalNum != 0 {
			return float64((snap.dealNum)*100.0) / float64(m.totalNum)
		}
		return 100
	}
	return 0
}
