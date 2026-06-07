package db_test

import (
	"os"
	"testing"

	"july/internal/testutil"
)

func TestMain(m *testing.M) {
	if err := testutil.SetupSharedEnv(); err != nil {
		_, _ = os.Stderr.WriteString("shared setup failed: " + err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(m.Run())
}
