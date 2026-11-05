package process

import (
	"strings"
	"testing"
)

func TestLimitedBufferBoundsMemoryAndMarksTruncation(t *testing.T) {
	b := &limitedBuffer{limit: 8}
	input := strings.Repeat("x", 1024*1024)
	n, err := b.Write([]byte(input))
	if err != nil || n != len(input) {
		t.Fatalf("writer must consume the child output while retaining only the cap: n=%d err=%v", n, err)
	}
	if b.buf.Len() != 8 {
		t.Fatalf("retained %d bytes, want 8", b.buf.Len())
	}
	if got := b.Result(); got != "xxxxxxxx…(truncated)" {
		t.Fatalf("unexpected truncated result %q", got)
	}
}
