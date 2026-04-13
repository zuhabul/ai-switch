package model

import "time"

type Profile struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	Frontend       string    `json:"frontend"`
	AuthMethod     string    `json:"auth_method"`
	Protocol       string    `json:"protocol"`
	Account        string    `json:"account"`
	OwnerScopes    []string  `json:"owner_scopes,omitempty"`
	Priority       int       `json:"priority"`
	Enabled        bool      `json:"enabled"`
	Tags           []string  `json:"tags,omitempty"`
	BudgetDailyUSD float64   `json:"budget_daily_usd,omitempty"`
	CooldownUntil  time.Time `json:"cooldown_until,omitempty"`
}

type HealthSnapshot struct {
	ProfileID              string    `json:"profile_id"`
	RemainingRequests5Min  int       `json:"remaining_requests_5min"`
	RemainingRequestsHour  int       `json:"remaining_requests_hour"`
	EstimatedLatencyMS     int       `json:"estimated_latency_ms"`
	RecentErrorRatePercent float64   `json:"recent_error_rate_percent"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type PolicyRule struct {
	Name               string   `json:"name"`
	Priority           int      `json:"priority"`
	Frontends          []string `json:"frontends,omitempty"`
	TaskClasses        []string `json:"task_classes,omitempty"`
	AllowProviders     []string `json:"allow_providers,omitempty"`
	DenyProviders      []string `json:"deny_providers,omitempty"`
	RequireAnyTag      []string `json:"require_any_tag,omitempty"`
	MaxBudgetDailyUSD  float64  `json:"max_budget_daily_usd,omitempty"`
	RequireAuthMethods []string `json:"require_auth_methods,omitempty"`
}

type Lease struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Frontend  string    `json:"frontend"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type TaskRequest struct {
	Frontend           string   `json:"frontend"`
	TaskClass          string   `json:"task_class"`
	RequiredProtocol   string   `json:"required_protocol,omitempty"`
	PreferredProviders []string `json:"preferred_providers,omitempty"`
	RequireTags        []string `json:"require_tags,omitempty"`
	Owner              string   `json:"owner,omitempty"`
}

type RouteDecision struct {
	ProfileID string   `json:"profile_id,omitempty"`
	Score     float64  `json:"score"`
	Reasons   []string `json:"reasons,omitempty"`
	Rejected  []string `json:"rejected,omitempty"`
}

type RoutePlan struct {
	Primary    RouteDecision   `json:"primary"`
	Candidates []RouteDecision `json:"candidates,omitempty"`
	Rejected   []string        `json:"rejected,omitempty"`
}

type Incident struct {
	ID              string    `json:"id"`
	ProfileID       string    `json:"profile_id"`
	Kind            string    `json:"kind"`
	Message         string    `json:"message,omitempty"`
	Owner           string    `json:"owner,omitempty"`
	CooldownSeconds int       `json:"cooldown_seconds,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type RuntimePlanRequest struct {
	Frontend           string   `json:"frontend"`
	TaskClass          string   `json:"task_class"`
	RequiredProtocol   string   `json:"required_protocol,omitempty"`
	PreferredProviders []string `json:"preferred_providers,omitempty"`
	RequireTags        []string `json:"require_tags,omitempty"`
	Owner              string   `json:"owner,omitempty"`
	Cwd                string   `json:"cwd,omitempty"`
	Model              string   `json:"model,omitempty"`
	Prompt             string   `json:"prompt,omitempty"`
	CommandArgs        []string `json:"command_args,omitempty"`
	LeaseTTLSeconds    int      `json:"lease_ttl_seconds,omitempty"`
}

type RuntimePlan struct {
	ProfileID string            `json:"profile_id"`
	LeaseID   string            `json:"lease_id"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env,omitempty"`
	Reasons   []string          `json:"reasons,omitempty"`
}

type DashboardProfile struct {
	Profile      Profile         `json:"profile"`
	Health       *HealthSnapshot `json:"health,omitempty"`
	Lease        *Lease          `json:"lease,omitempty"`
	SecretCount  int             `json:"secret_count"`
	LastHealthAt *time.Time      `json:"last_health_at,omitempty"`
}

type DashboardAccount struct {
	Provider        string   `json:"provider"`
	Account         string   `json:"account"`
	ProfileIDs      []string `json:"profile_ids"`
	Frontends       []string `json:"frontends"`
	ActiveLeases    int      `json:"active_leases"`
	HealthyProfiles int      `json:"healthy_profiles"`
}

type DashboardSummary struct {
	TimeUTC         time.Time          `json:"time_utc"`
	Counts          map[string]int     `json:"counts"`
	Providers       map[string]int     `json:"providers"`
	Profiles        []DashboardProfile `json:"profiles"`
	Accounts        []DashboardAccount `json:"accounts"`
	Policies        []PolicyRule       `json:"policies"`
	ActiveLeases    []Lease            `json:"active_leases"`
	RecentIncidents []Incident         `json:"recent_incidents,omitempty"`
}

type State struct {
	Profiles       map[string]Profile           `json:"profiles"`
	Health         map[string]HealthSnapshot    `json:"health"`
	Policies       []PolicyRule                 `json:"policies"`
	Leases         map[string]Lease             `json:"leases"`
	Incidents      []Incident                   `json:"incidents,omitempty"`
	SecretBindings map[string]map[string]string `json:"secret_bindings"` // profile_id -> env_var -> secret_key
}

func NewState() State {
	return State{
		Profiles:       map[string]Profile{},
		Health:         map[string]HealthSnapshot{},
		Policies:       []PolicyRule{},
		Leases:         map[string]Lease{},
		Incidents:      []Incident{},
		SecretBindings: map[string]map[string]string{},
	}
}
