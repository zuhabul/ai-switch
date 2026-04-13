package adapter

import "slices"

type Capabilities struct {
	Provider             string   `json:"provider"`
	Frontend             string   `json:"frontend"`
	AuthMethods          []string `json:"auth_methods"`
	Protocols            []string `json:"protocols"`
	SupportsLeases       bool     `json:"supports_leases"`
	SupportsResume       bool     `json:"supports_resume"`
	SupportsFailover     bool     `json:"supports_failover"`
	SupportsMultiAccount bool     `json:"supports_multi_account"`
}

type Registry struct {
	entries map[string]Capabilities
}

func NewRegistry() *Registry {
	r := &Registry{entries: map[string]Capabilities{}}
	for _, c := range builtins() {
		r.entries[key(c.Provider, c.Frontend)] = c
	}
	return r
}

func (r *Registry) Get(provider, frontend string) (Capabilities, bool) {
	c, ok := r.entries[key(provider, frontend)]
	return c, ok
}

func (r *Registry) List() []Capabilities {
	out := make([]Capabilities, 0, len(r.entries))
	for _, c := range r.entries {
		out = append(out, c)
	}
	slices.SortStableFunc(out, func(a, b Capabilities) int {
		if a.Provider < b.Provider {
			return -1
		}
		if a.Provider > b.Provider {
			return 1
		}
		if a.Frontend < b.Frontend {
			return -1
		}
		if a.Frontend > b.Frontend {
			return 1
		}
		return 0
	})
	return out
}

func key(provider, frontend string) string {
	return provider + "::" + frontend
}

func builtins() []Capabilities {
	return []Capabilities{
		{Provider: "openai", Frontend: "codex", AuthMethods: []string{"chatgpt", "api_key"}, Protocols: []string{"native_cli", "app_server", "openai_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "openai", Frontend: "opencode", AuthMethods: []string{"chatgpt", "api_key"}, Protocols: []string{"native_cli", "openai_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "openai", Frontend: "openclaw", AuthMethods: []string{"chatgpt", "api_key"}, Protocols: []string{"native_cli", "openai_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "anthropic", Frontend: "claude_code", AuthMethods: []string{"claude_app", "api_key", "bedrock", "vertex"}, Protocols: []string{"native_cli", "anthropic_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "google", Frontend: "gemini_cli", AuthMethods: []string{"google_login", "api_key", "vertex"}, Protocols: []string{"native_cli", "gemini_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "google", Frontend: "aider", AuthMethods: []string{"api_key"}, Protocols: []string{"native_cli", "openai_compatible", "anthropic_compatible", "gemini_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "alibaba", Frontend: "qwen_code", AuthMethods: []string{"oauth", "api_key"}, Protocols: []string{"native_cli", "openai_compatible", "anthropic_compatible", "gemini_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "moonshot", Frontend: "kimi_cli", AuthMethods: []string{"oauth", "api_key"}, Protocols: []string{"native_cli", "acp", "openai_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "github", Frontend: "copilot", AuthMethods: []string{"oauth", "copilot_subscription"}, Protocols: []string{"native_cli", "openai_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "xai", Frontend: "grok", AuthMethods: []string{"api_key"}, Protocols: []string{"openai_compatible", "native_cli"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "minimax", Frontend: "hermes", AuthMethods: []string{"api_key"}, Protocols: []string{"openai_compatible", "anthropic_compatible", "hermes", "native_cli"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "zai", Frontend: "coding_tool_helper", AuthMethods: []string{"api_key"}, Protocols: []string{"helper", "openai_compatible"}, SupportsLeases: true, SupportsResume: false, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "openrouter", Frontend: "openrouter_cli", AuthMethods: []string{"api_key"}, Protocols: []string{"openai_compatible"}, SupportsLeases: true, SupportsResume: false, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "deepseek", Frontend: "deepseek_cli", AuthMethods: []string{"api_key"}, Protocols: []string{"openai_compatible"}, SupportsLeases: true, SupportsResume: false, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "mistral", Frontend: "mistral_cli", AuthMethods: []string{"api_key", "oauth"}, Protocols: []string{"openai_compatible", "native_cli"}, SupportsLeases: true, SupportsResume: false, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "sourcegraph", Frontend: "cody_cli", AuthMethods: []string{"oauth", "api_key"}, Protocols: []string{"native_cli"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "continue", Frontend: "continue_cli", AuthMethods: []string{"api_key", "oauth"}, Protocols: []string{"native_cli", "openai_compatible", "anthropic_compatible", "gemini_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "cursor", Frontend: "cursor_agent", AuthMethods: []string{"oauth", "api_key"}, Protocols: []string{"native_cli", "openai_compatible", "anthropic_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
		{Provider: "codeium", Frontend: "windsurf", AuthMethods: []string{"oauth", "api_key"}, Protocols: []string{"native_cli", "openai_compatible", "anthropic_compatible"}, SupportsLeases: true, SupportsResume: true, SupportsFailover: true, SupportsMultiAccount: true},
	}
}
