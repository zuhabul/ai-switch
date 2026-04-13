package adapter

import "testing"

func TestRegistryHasCoreProviders(t *testing.T) {
	r := NewRegistry()
	cases := [][2]string{
		{"openai", "codex"},
		{"anthropic", "claude_code"},
		{"google", "gemini_cli"},
		{"alibaba", "qwen_code"},
		{"moonshot", "kimi_cli"},
		{"minimax", "hermes"},
		{"xai", "grok"},
	}
	for _, c := range cases {
		if _, ok := r.Get(c[0], c[1]); !ok {
			t.Fatalf("missing adapter %s/%s", c[0], c[1])
		}
	}
}
