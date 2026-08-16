package version

import (
	"strings"
	"testing"
)

func TestStringContainsVersion(t *testing.T) {
	s := String()
	if !strings.Contains(s, Version) {
		t.Errorf("String() = %q, want it to contain version %q", s, Version)
	}
	if !strings.Contains(s, "CYBEROS") {
		t.Errorf("String() = %q, want it to contain CYBEROS", s)
	}
}

func TestInfo(t *testing.T) {
	info := Info()
	if info["version"] != Version {
		t.Errorf("Info()[version] = %q, want %q", info["version"], Version)
	}
}
