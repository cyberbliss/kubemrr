package app

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	enableVeryVerbose()
	code := m.Run()
	os.Exit(code)
}
