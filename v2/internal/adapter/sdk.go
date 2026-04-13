package adapter

import "fmt"

type LaunchRequest struct {
	Frontend string
	Model    string
	Prompt   string
	Cwd      string
	Args     []string
}

type LaunchSpec struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type RuntimeHook interface {
	Frontend() string
	Build(req LaunchRequest) (LaunchSpec, error)
}

type HookRegistry struct {
	hooks map[string]RuntimeHook
}

func NewHookRegistry() *HookRegistry {
	r := &HookRegistry{hooks: map[string]RuntimeHook{}}
	for _, h := range builtinHooks() {
		r.hooks[h.Frontend()] = h
	}
	return r
}

func (r *HookRegistry) Get(frontend string) (RuntimeHook, bool) {
	h, ok := r.hooks[frontend]
	return h, ok
}

func BuildDefault(frontend string, req LaunchRequest) (LaunchSpec, error) {
	hooks := NewHookRegistry()
	h, ok := hooks.Get(frontend)
	if !ok {
		if frontend == "" {
			return LaunchSpec{}, fmt.Errorf("frontend is required")
		}
		// Generic passthrough for unknown frontends.
		return LaunchSpec{Command: frontend, Args: req.Args, Env: map[string]string{}}, nil
	}
	return h.Build(req)
}
