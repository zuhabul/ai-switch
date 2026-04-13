package adapter

import "testing"

func TestBuildDefaultKnownHook(t *testing.T) {
	spec, err := BuildDefault("codex", LaunchRequest{Model: "gpt-5", Args: nil})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.Command != "codex" {
		t.Fatalf("command %q", spec.Command)
	}
	if len(spec.Args) == 0 || spec.Args[0] != "app-server" {
		t.Fatalf("unexpected args: %v", spec.Args)
	}
}

func TestBuildDefaultFallback(t *testing.T) {
	spec, err := BuildDefault("hermes", LaunchRequest{Args: []string{"acp"}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.Command != "hermes" {
		t.Fatalf("command %q", spec.Command)
	}
}

func TestBuildDefaultNativeHookOpencode(t *testing.T) {
	spec, err := BuildDefault("opencode", LaunchRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.Command != "opencode" {
		t.Fatalf("command %q", spec.Command)
	}
	if len(spec.Args) == 0 || spec.Args[0] != "-p" {
		t.Fatalf("unexpected args: %v", spec.Args)
	}
}

func TestHookRegistryListFrontends(t *testing.T) {
	r := NewHookRegistry()
	items := r.ListFrontends()
	if len(items) < 8 {
		t.Fatalf("expected expanded hooks, got %d", len(items))
	}
}
