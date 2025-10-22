package sls

import (
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

func TestPB(t *testing.T) {
	lg := &LogGroup{
		Logs: []*Log{{
			Time: proto.Uint32(1234567890),
		}},
	}
	_, err := proto.Marshal(lg)
	if err != nil {
		t.Errorf("proto.Marshal error: %v", err)
	}
}

func TestLoadingFromJson(t *testing.T) {
	// 原始 JSON 数据
	jsonData := `{
        "Logs": [
            {
                "Time": 1712345678,
                "Contents": [
                    { "Key": "method", "Value": "POST" },
                    { "Key": "path", "Value": "/api/login" }
                ]
            }
        ],
        "Topic": "web-logs",
        "Source": "192.168.1.100",
        "LogTags": [
            { "Key": "env", "Value": "prod" }
        ]
    }`

	// 创建空的 LogGroup 对象
	logGroup := &LogGroup{}

	// 使用 jsonpb 解析 JSON 到 Protobuf 结构体
	err := jsonpb.UnmarshalString(jsonData, logGroup)
	if err != nil {
		t.Error("Failed to unmarshal JSON to protobuf: ", err)
	}
	t.Logf("LogGroup: %+v\n", logGroup)

	data, _ := proto.Marshal(logGroup)
	t.Logf("Serialized size: %d bytes\n", len(data))
}
