package machine

import (
	"os"
	"testing"
)

// The historical BASIC tapes are external fixtures, not part of this repository.
// An explicitly configured directory must contain the files required by each test.
func kimProgramsDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("M65C02_KIM_PROGRAMS")
	if dir == "" {
		t.Skip("set M65C02_KIM_PROGRAMS to the external KIM BASIC tape directory")
	}
	if info, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("external KIM BASIC tapes unavailable: %s", dir)
	} else if err != nil {
		t.Fatal(err)
	} else if !info.IsDir() {
		t.Fatalf("KIM BASIC tape path is not a directory: %s", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Skipf("external KIM BASIC tape directory is empty: %s", dir)
	}
	return dir
}
