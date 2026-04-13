package vault

import (
	"path/filepath"
	"testing"
)

func TestFileVaultRoundTrip(t *testing.T) {
	t.Setenv("AISWITCH_MASTER_KEY", "test-key")
	v := NewFileVault(filepath.Join(t.TempDir(), "secrets.enc.json"))
	if err := v.Set("openai_key", "sk-test-123"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := v.Get("openai_key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "sk-test-123" {
		t.Fatalf("got %q", got)
	}
	names, err := v.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "openai_key" {
		t.Fatalf("unexpected names: %v", names)
	}
	if err := v.Delete("openai_key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := v.Get("openai_key"); err == nil {
		t.Fatalf("expected not found error")
	}
}
