package cli

import (
	"testing"

	"cyberos/internal/sdk"
)

func TestStatusDoctor(t *testing.T) {
	app := New(sdk.NewInMemory())
	if code := app.Run([]string{"status"}); code != 0 {
		t.Fatal(code)
	}
	// doctor may report degraded if Control not wired in NewInMemory
	code := app.Run([]string{"doctor"})
	if code != 0 && code != 5 {
		t.Fatal(code)
	}
	if code := app.Run([]string{"trust", "help"}); code != 0 {
		t.Fatal(code)
	}
}
