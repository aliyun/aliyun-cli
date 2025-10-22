package sls

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestParseProtobufList(t *testing.T) {
	t.Run("EmptyData", func(t *testing.T) {
		result, err := ParseProtobufList([]byte{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("ValidLogGroupList", func(t *testing.T) {
		// Create a sample LogGroup
		logGroup := &LogGroup{
			Logs: []*Log{{
				Time: proto.Uint32(1234567890),
			}},
			Topic: proto.String("test-topic"),
		}

		// Create a LogGroupList with the LogGroup
		logGroupList := &LogGroupList{
			LogGroups: []*LogGroup{logGroup},
		}

		// Marshal to protobuf bytes
		data, err := proto.Marshal(logGroupList)
		assert.NoError(t, err)

		// Test parsing
		result, err := ParseProtobufList(data)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "test-topic", *result[0].Topic)
	})

	t.Run("InvalidData", func(t *testing.T) {
		invalidData := []byte("invalid protobuf data")
		result, err := ParseProtobufList(invalidData)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot parse proto data")
	})
}

func TestProcessPullLogsResponse(t *testing.T) {
	t.Run("ValidLogGroups", func(t *testing.T) {
		// Create sample LogGroups
		logGroups := []*LogGroup{
			{
				Logs: []*Log{{
					Time: proto.Uint32(1234567890),
				}},
				Topic: proto.String("test-topic"),
			},
		}

		// Create LogGroupList and marshal to bytes
		logGroupList := &LogGroupList{LogGroups: logGroups}
		bodyBytes, err := proto.Marshal(logGroupList)
		assert.NoError(t, err)

		// Test processing
		result, err := ProcessPullLogsResponse(bodyBytes)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify the result is valid JSON
		var parsedResult LogGroupList
		err = json.Unmarshal(result, &parsedResult)
		assert.NoError(t, err)
		assert.Len(t, parsedResult.LogGroups, 1)
		assert.Equal(t, "test-topic", *parsedResult.LogGroups[0].Topic)
	})

	t.Run("InvalidData", func(t *testing.T) {
		invalidData := []byte("invalid data")
		result, err := ProcessPullLogsResponse(invalidData)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "parse proto object failed")
	})
}

func TestCompressLZ4(t *testing.T) {
	t.Run("ValidData", func(t *testing.T) {
		data := []byte("test data for compression")
		result, err := CompressLZ4(data)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, 0, len(result))
	})

	t.Run("EmptyData", func(t *testing.T) {
		data := []byte{}
		result, err := CompressLZ4(data)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "data cannot be empty")
	})
}

func TestPreparePutLogsData(t *testing.T) {
	t.Run("ValidJSON", func(t *testing.T) {
		// Create sample JSON data for a LogGroup
		logGroup := LogGroup{
			Logs: []*Log{
				{
					Time: proto.Uint32(1234567890),
					Contents: []*LogContent{
						{Key: proto.String("key1"), Value: proto.String("value1")},
					},
				},
			},
			Topic: proto.String("test-topic"),
		}

		bodyBytes, err := json.Marshal(logGroup)
		assert.NoError(t, err)

		// Test preparation
		compressedData, rawSize, err := PreparePutLogsData(bodyBytes)
		assert.NoError(t, err)
		assert.NotNil(t, compressedData)
		assert.NotEqual(t, 0, rawSize)
		assert.NotEqual(t, 0, len(compressedData))
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := []byte("invalid json")
		compressedData, rawSize, err := PreparePutLogsData(invalidJSON)
		assert.Error(t, err)
		assert.Nil(t, compressedData)
		assert.Equal(t, 0, rawSize)
		assert.Contains(t, err.Error(), "parse json failed")
	})

	t.Run("EmptyLogs", func(t *testing.T) {
		// Create JSON data with empty logs
		logGroup := LogGroup{
			Logs: []*Log{}, // Empty logs
		}

		bodyBytes, err := json.Marshal(logGroup)
		assert.NoError(t, err)

		// Test preparation
		compressedData, rawSize, err := PreparePutLogsData(bodyBytes)
		assert.Error(t, err)
		assert.Nil(t, compressedData)
		assert.Equal(t, 0, rawSize)
		assert.Contains(t, err.Error(), "log cannot be empty")
	})
}
