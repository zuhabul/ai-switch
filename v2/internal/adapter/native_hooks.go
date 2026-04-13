package adapter

import "strings"

type codexHook struct{}
type claudeHook struct{}
type geminiHook struct{}
type simpleHook struct {
	frontend       string
	command        string
	modelEnv       string
	defaultWithP   bool
	defaultCommand []string
}

func builtinHooks() []RuntimeHook {
	return []RuntimeHook{
		codexHook{},
		claudeHook{},
		geminiHook{},
		simpleHook{frontend: "opencode", command: "opencode", modelEnv: "OPENCODE_MODEL", defaultWithP: true},
		simpleHook{frontend: "openclaw", command: "openclaw", modelEnv: "OPENCLAW_MODEL", defaultWithP: true},
		simpleHook{frontend: "qwen_code", command: "qwen-code", modelEnv: "QWEN_MODEL", defaultWithP: true},
		simpleHook{frontend: "kimi_cli", command: "kimi", modelEnv: "KIMI_MODEL", defaultWithP: true},
		simpleHook{frontend: "copilot", command: "copilot", modelEnv: "COPILOT_MODEL", defaultWithP: true},
		simpleHook{frontend: "aider", command: "aider", modelEnv: "AIDER_MODEL", defaultWithP: false},
		simpleHook{frontend: "cody_cli", command: "cody", modelEnv: "CODY_MODEL", defaultWithP: true},
		simpleHook{frontend: "continue_cli", command: "continue", modelEnv: "CONTINUE_MODEL", defaultWithP: true},
		simpleHook{frontend: "cursor_agent", command: "cursor-agent", modelEnv: "CURSOR_MODEL", defaultWithP: true},
		simpleHook{frontend: "windsurf", command: "windsurf", modelEnv: "WINDSURF_MODEL", defaultWithP: true},
		simpleHook{frontend: "openrouter_cli", command: "openrouter", modelEnv: "OPENROUTER_MODEL", defaultWithP: true},
		simpleHook{frontend: "deepseek_cli", command: "deepseek", modelEnv: "DEEPSEEK_MODEL", defaultWithP: true},
		simpleHook{frontend: "mistral_cli", command: "mistral", modelEnv: "MISTRAL_MODEL", defaultWithP: true},
		simpleHook{frontend: "hermes", command: "hermes", modelEnv: "HERMES_MODEL", defaultWithP: true},
		simpleHook{frontend: "grok", command: "grok", modelEnv: "GROK_MODEL", defaultWithP: true},
	}
}

func (codexHook) Frontend() string { return "codex" }

func (codexHook) Build(req LaunchRequest) (LaunchSpec, error) {
	args := append([]string{}, req.Args...)
	if len(args) == 0 {
		args = []string{"app-server", "--listen", "stdio://"}
	}
	spec := LaunchSpec{Command: "codex", Args: args, Env: map[string]string{}}
	if strings.TrimSpace(req.Model) != "" {
		spec.Env["CODEX_MODEL"] = strings.TrimSpace(req.Model)
	}
	return spec, nil
}

func (claudeHook) Frontend() string { return "claude_code" }

func (claudeHook) Build(req LaunchRequest) (LaunchSpec, error) {
	args := append([]string{}, req.Args...)
	if len(args) == 0 {
		args = []string{"-p", req.Prompt}
	}
	spec := LaunchSpec{Command: "claude", Args: args, Env: map[string]string{}}
	if strings.TrimSpace(req.Model) != "" {
		spec.Env["CLAUDE_MODEL"] = strings.TrimSpace(req.Model)
	}
	return spec, nil
}

func (geminiHook) Frontend() string { return "gemini_cli" }

func (geminiHook) Build(req LaunchRequest) (LaunchSpec, error) {
	args := append([]string{}, req.Args...)
	if len(args) == 0 {
		args = []string{"-p", req.Prompt, "--yolo", "-o", "text"}
	}
	spec := LaunchSpec{Command: "gemini", Args: args, Env: map[string]string{}}
	if strings.TrimSpace(req.Model) != "" {
		spec.Env["GEMINI_MODEL"] = strings.TrimSpace(req.Model)
	}
	return spec, nil
}

func (h simpleHook) Frontend() string { return h.frontend }

func (h simpleHook) Build(req LaunchRequest) (LaunchSpec, error) {
	args := append([]string{}, req.Args...)
	if len(args) == 0 {
		switch {
		case len(h.defaultCommand) > 0:
			args = append([]string{}, h.defaultCommand...)
		case h.frontend == "aider" && strings.TrimSpace(req.Prompt) != "":
			args = []string{req.Prompt}
		case h.defaultWithP && strings.TrimSpace(req.Prompt) != "":
			args = []string{"-p", req.Prompt}
		}
	}
	spec := LaunchSpec{Command: h.command, Args: args, Env: map[string]string{}}
	if strings.TrimSpace(req.Model) != "" && strings.TrimSpace(h.modelEnv) != "" {
		spec.Env[h.modelEnv] = strings.TrimSpace(req.Model)
	}
	return spec, nil
}
