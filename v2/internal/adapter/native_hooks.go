package adapter

import "strings"

type codexHook struct{}
type claudeHook struct{}
type geminiHook struct{}

func builtinHooks() []RuntimeHook {
	return []RuntimeHook{
		codexHook{},
		claudeHook{},
		geminiHook{},
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
