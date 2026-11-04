//go:build !darwin && !linux

package main

import "fmt"

func restoreWriteFailureLimit() error {
	return fmt.Errorf("file-size limit tests require Darwin or Linux")
}
