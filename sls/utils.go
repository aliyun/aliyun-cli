package sls

import (
	"encoding/json"
	fmt "fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pierrec/lz4"
)

func ParseProtobufList(data []byte) ([]*LogGroup, error) {
	if len(data) == 0 {
		return nil, nil
	}

	logGroupList := &LogGroupList{}
	if err := proto.Unmarshal(data, logGroupList); err == nil && len(logGroupList.LogGroups) > 0 {
		return logGroupList.LogGroups, nil
	}

	return nil, fmt.Errorf("cannot parse proto data: no LogGroup or LogGroupList")
}

func ProcessPullLogsResponse(bodyBytes []byte) ([]byte, error) {
	logGroups, err := ParseProtobufList(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse proto object failed: %v", err)
	}
	logGroupList := &LogGroupList{
		LogGroups: logGroups,
	}

	jsonData, err := json.Marshal(logGroupList)
	if err != nil {
		return nil, fmt.Errorf("serilization failed: %v", err)
	}

	return jsonData, nil
}

func CompressLZ4(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, fmt.Errorf("lz4 compress failed: %v", err)
	}
	return buf[:n], nil
}

func PreparePutLogsData(bodyBytes []byte) (compressedData []byte, rawSize int, err error) {
	var logGroup LogGroup
	if err = json.Unmarshal(bodyBytes, &logGroup); err != nil {
		return nil, 0, fmt.Errorf("parse json failed: %v", err)
	}
	if len(logGroup.Logs) == 0 {
		return nil, 0, fmt.Errorf("log cannot be empty, please check")
	}
	protobufData, err := proto.Marshal(&logGroup)
	if err != nil {
		return nil, 0, fmt.Errorf("serialize pb failed: %v", err)
	}

	rawSize = len(protobufData)
	compressedData, err = CompressLZ4(protobufData)
	if err != nil {
		return nil, 0, fmt.Errorf("lz4 compress failed: %v", err)
	}

	return compressedData, rawSize, nil
}
