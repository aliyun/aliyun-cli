package util

import (
	"net/http"
	"os"
	"time"
)

func GetFromEnv(args ...string) string {
	for _, key := range args {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return ""
}

func GetCurrentUnixTime() int64 {
	return time.Now().Unix()
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}
