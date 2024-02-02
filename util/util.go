package util

import "os"

func GetFromEnv(args ...string) string {
	for _, key := range args {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return ""
}
