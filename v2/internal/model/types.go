package model

import "time"

type Profile struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	Frontend       string    `json:"frontend"`
	AuthMethod     string    `json:"auth_method"`
	Protocol       string    `json:"protocol"`
	Account        string    `json:"account"`
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

type State struct {
	Profiles map[string]Profile        `json:"profiles"`
	Health   map[string]HealthSnapshot `json:"health"`
	Policies []PolicyRule              `json:"policies"`
	Leases   map[string]Lease          `json:"leases"`
}

func NewState() State {
	return State{
		Profiles: map[string]Profile{},
		Health:   map[string]HealthSnapshot{},
		Policies: []PolicyRule{},
		Leases:   map[string]Lease{},
	}
}
