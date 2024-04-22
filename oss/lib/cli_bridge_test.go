package lib

import (
	"testing"
)

func TestCliBridge(t *testing.T) {
	NewCommandBridge(configCommand.command)
}
