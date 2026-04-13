package adapter

type ContractMethod struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	InputFields []string `json:"input_fields,omitempty"`
	Output      string   `json:"output,omitempty"`
}

type ContractSpec struct {
	Version string           `json:"version"`
	Name    string           `json:"name"`
	Methods []ContractMethod `json:"methods"`
}

func DefaultContract() ContractSpec {
	return ContractSpec{
		Version: "v1alpha1",
		Name:    "ai-switch-adapter-contract",
		Methods: []ContractMethod{
			{
				Name:        "detect",
				Description: "Detect whether frontend/provider tooling is installed and reachable.",
				InputFields: []string{"frontend", "provider"},
				Output:      "AdapterCapabilities",
			},
			{
				Name:        "validate",
				Description: "Validate profile/auth/protocol compatibility before routing.",
				InputFields: []string{"profile"},
				Output:      "ValidationResult",
			},
			{
				Name:        "refresh",
				Description: "Refresh account/session auth state for the given profile.",
				InputFields: []string{"profile_id"},
				Output:      "RefreshResult",
			},
			{
				Name:        "launch",
				Description: "Build command/env launch spec for runtime execution.",
				InputFields: []string{"launch_request"},
				Output:      "LaunchSpec",
			},
			{
				Name:        "checkpoint",
				Description: "Persist resumable runtime/session checkpoint.",
				InputFields: []string{"profile_id", "runtime_id", "checkpoint_payload"},
				Output:      "CheckpointReference",
			},
			{
				Name:        "resume",
				Description: "Resume runtime from checkpoint or known session id.",
				InputFields: []string{"profile_id", "checkpoint_id"},
				Output:      "LaunchSpec",
			},
		},
	}
}
