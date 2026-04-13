package service

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/router"
)

type Store interface {
	Load() (model.State, error)
	Save(model.State) error
}

type Service struct {
	store Store
	mu    sync.Mutex
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Init(ctx context.Context) error {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return err
	}
	return s.store.Save(state)
}

func (s *Service) AddProfile(ctx context.Context, p model.Profile) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if err := validateProfile(p); err != nil {
		return err
	}
	state.Profiles[p.ID] = p
	return s.store.Save(state)
}

func (s *Service) UpdateProfile(ctx context.Context, p model.Profile) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if err := validateProfile(p); err != nil {
		return err
	}
	if _, ok := state.Profiles[p.ID]; !ok {
		return fmt.Errorf("unknown profile %s", p.ID)
	}
	state.Profiles[p.ID] = p
	return s.store.Save(state)
}

func (s *Service) DeleteProfile(ctx context.Context, profileID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if _, ok := state.Profiles[profileID]; !ok {
		return fmt.Errorf("unknown profile %s", profileID)
	}
	now := time.Now().UTC()
	for _, l := range state.Leases {
		if l.ProfileID == profileID && l.ExpiresAt.After(now) {
			return fmt.Errorf("profile %s has active lease %s owned by %s", profileID, l.ID, l.Owner)
		}
	}
	delete(state.Profiles, profileID)
	delete(state.Health, profileID)
	delete(state.SecretBindings, profileID)
	return s.store.Save(state)
}

func (s *Service) ListProfiles(ctx context.Context) ([]model.Profile, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]model.Profile, 0, len(state.Profiles))
	for _, p := range state.Profiles {
		out = append(out, p)
	}
	slices.SortStableFunc(out, func(a, b model.Profile) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return out, nil
}

func (s *Service) GetProfile(ctx context.Context, profileID string) (model.Profile, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return model.Profile{}, err
	}
	p, ok := state.Profiles[profileID]
	if !ok {
		return model.Profile{}, fmt.Errorf("unknown profile %s", profileID)
	}
	return p, nil
}

func (s *Service) AddPolicy(ctx context.Context, rule model.PolicyRule) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	state.Policies = append(state.Policies, rule)
	return s.store.Save(state)
}

func (s *Service) UpdatePolicy(ctx context.Context, rule model.PolicyRule) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if rule.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	idx := -1
	for i := range state.Policies {
		if state.Policies[i].Name == rule.Name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("unknown policy %s", rule.Name)
	}
	state.Policies[idx] = rule
	return s.store.Save(state)
}

func (s *Service) DeletePolicy(ctx context.Context, name string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("policy name is required")
	}
	idx := -1
	for i := range state.Policies {
		if state.Policies[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("unknown policy %s", name)
	}
	state.Policies = append(state.Policies[:idx], state.Policies[idx+1:]...)
	return s.store.Save(state)
}

func (s *Service) ListPolicies(ctx context.Context) ([]model.PolicyRule, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := slices.Clone(state.Policies)
	slices.SortStableFunc(out, func(a, b model.PolicyRule) int {
		if a.Priority > b.Priority {
			return -1
		}
		if a.Priority < b.Priority {
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out, nil
}

func (s *Service) GetPolicy(ctx context.Context, name string) (model.PolicyRule, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return model.PolicyRule{}, err
	}
	for _, p := range state.Policies {
		if p.Name == name {
			return p, nil
		}
	}
	return model.PolicyRule{}, fmt.Errorf("unknown policy %s", name)
}

func (s *Service) BindSecret(ctx context.Context, profileID, envVar, secretKey string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if _, ok := state.Profiles[profileID]; !ok {
		return fmt.Errorf("unknown profile %s", profileID)
	}
	if envVar == "" || secretKey == "" {
		return fmt.Errorf("env var and secret key are required")
	}
	if state.SecretBindings[profileID] == nil {
		state.SecretBindings[profileID] = map[string]string{}
	}
	state.SecretBindings[profileID][envVar] = secretKey
	return s.store.Save(state)
}

func (s *Service) UnbindSecret(ctx context.Context, profileID, envVar string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if _, ok := state.Profiles[profileID]; !ok {
		return fmt.Errorf("unknown profile %s", profileID)
	}
	if state.SecretBindings[profileID] == nil {
		return fmt.Errorf("no bindings for profile %s", profileID)
	}
	if _, ok := state.SecretBindings[profileID][envVar]; !ok {
		return fmt.Errorf("binding %s not found for profile %s", envVar, profileID)
	}
	delete(state.SecretBindings[profileID], envVar)
	if len(state.SecretBindings[profileID]) == 0 {
		delete(state.SecretBindings, profileID)
	}
	return s.store.Save(state)
}

func (s *Service) ListSecretBindings(ctx context.Context, profileID string) (map[string]string, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if _, ok := state.Profiles[profileID]; !ok {
		return nil, fmt.Errorf("unknown profile %s", profileID)
	}
	out := map[string]string{}
	for k, v := range state.SecretBindings[profileID] {
		out[k] = v
	}
	return out, nil
}

func (s *Service) UpdateHealth(ctx context.Context, snapshot model.HealthSnapshot) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if _, ok := state.Profiles[snapshot.ProfileID]; !ok {
		return fmt.Errorf("unknown profile %s", snapshot.ProfileID)
	}
	if snapshot.UpdatedAt.IsZero() {
		snapshot.UpdatedAt = time.Now().UTC()
	}
	state.Health[snapshot.ProfileID] = snapshot
	return s.store.Save(state)
}

func (s *Service) ListHealth(ctx context.Context) (map[string]model.HealthSnapshot, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make(map[string]model.HealthSnapshot, len(state.Health))
	for k, v := range state.Health {
		out[k] = v
	}
	return out, nil
}

func (s *Service) SetCooldown(ctx context.Context, profileID string, d time.Duration) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	p, ok := state.Profiles[profileID]
	if !ok {
		return fmt.Errorf("unknown profile %s", profileID)
	}
	p.CooldownUntil = time.Now().UTC().Add(d)
	state.Profiles[p.ID] = p
	return s.store.Save(state)
}

func (s *Service) Route(ctx context.Context, req model.TaskRequest) (model.RouteDecision, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return model.RouteDecision{}, err
	}
	profiles := make([]model.Profile, 0, len(state.Profiles))
	for _, p := range state.Profiles {
		profiles = append(profiles, p)
	}
	decision := router.PickBest(router.Input{
		Profiles: profiles,
		Health:   state.Health,
		Policies: state.Policies,
		Now:      time.Now().UTC(),
		Request:  req,
	})
	if decision.ProfileID == "" {
		return decision, fmt.Errorf("no eligible profile found")
	}
	return decision, nil
}

func (s *Service) RoutePlan(ctx context.Context, req model.TaskRequest) (model.RoutePlan, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return model.RoutePlan{}, err
	}
	profiles := make([]model.Profile, 0, len(state.Profiles))
	for _, p := range state.Profiles {
		profiles = append(profiles, p)
	}
	candidates, rejected := router.Rank(router.Input{
		Profiles: profiles,
		Health:   state.Health,
		Policies: state.Policies,
		Now:      time.Now().UTC(),
		Request:  req,
	})
	if len(candidates) == 0 {
		return model.RoutePlan{
			Primary:  model.RouteDecision{Score: 0, Rejected: rejected},
			Rejected: rejected,
		}, fmt.Errorf("no eligible profile found")
	}
	primary := candidates[0]
	primary.Rejected = rejected
	return model.RoutePlan{
		Primary:    primary,
		Candidates: candidates,
		Rejected:   rejected,
	}, nil
}

func (s *Service) AcquireLease(ctx context.Context, profileID, frontend, owner string, ttl time.Duration) (model.Lease, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	state, err := s.store.Load()
	if err != nil {
		return model.Lease{}, err
	}
	if _, ok := state.Profiles[profileID]; !ok {
		return model.Lease{}, fmt.Errorf("unknown profile %s", profileID)
	}
	now := time.Now().UTC()
	for id, l := range state.Leases {
		if l.ProfileID == profileID && l.ExpiresAt.After(now) {
			if l.Owner == owner {
				// Idempotent refresh for the same owner avoids false conflicts
				// when previous runs crash before explicit lease release.
				l.Frontend = frontend
				l.ExpiresAt = now.Add(ttl)
				state.Leases[id] = l
				if err := s.store.Save(state); err != nil {
					return model.Lease{}, err
				}
				return l, nil
			}
			return model.Lease{}, fmt.Errorf("profile %s already leased by %s until %s", profileID, l.Owner, l.ExpiresAt.Format(time.RFC3339))
		}
	}
	lease := model.Lease{
		ID:        fmt.Sprintf("lease_%d", now.UnixNano()),
		ProfileID: profileID,
		Frontend:  frontend,
		Owner:     owner,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	state.Leases[lease.ID] = lease
	if err := s.store.Save(state); err != nil {
		return model.Lease{}, err
	}
	return lease, nil
}

func (s *Service) ReleaseLease(ctx context.Context, leaseID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	if _, ok := state.Leases[leaseID]; !ok {
		return fmt.Errorf("unknown lease %s", leaseID)
	}
	delete(state.Leases, leaseID)
	return s.store.Save(state)
}

func (s *Service) DashboardSummary(ctx context.Context) (model.DashboardSummary, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return model.DashboardSummary{}, err
	}

	now := time.Now().UTC()
	activeLeases := make([]model.Lease, 0, len(state.Leases))
	leaseByProfile := map[string]model.Lease{}
	for _, l := range state.Leases {
		if l.ExpiresAt.Before(now) {
			continue
		}
		activeLeases = append(activeLeases, l)
		leaseByProfile[l.ProfileID] = l
	}
	slices.SortStableFunc(activeLeases, func(a, b model.Lease) int {
		if a.ExpiresAt.Before(b.ExpiresAt) {
			return -1
		}
		if a.ExpiresAt.After(b.ExpiresAt) {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	profiles := make([]model.Profile, 0, len(state.Profiles))
	for _, p := range state.Profiles {
		profiles = append(profiles, p)
	}
	slices.SortStableFunc(profiles, func(a, b model.Profile) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	providers := map[string]int{}
	dashProfiles := make([]model.DashboardProfile, 0, len(profiles))
	for _, p := range profiles {
		providers[p.Provider]++
		dp := model.DashboardProfile{
			Profile:     p,
			SecretCount: len(state.SecretBindings[p.ID]),
		}
		if h, ok := state.Health[p.ID]; ok {
			hc := h
			dp.Health = &hc
			if !h.UpdatedAt.IsZero() {
				t := h.UpdatedAt
				dp.LastHealthAt = &t
			}
		}
		if l, ok := leaseByProfile[p.ID]; ok {
			lc := l
			dp.Lease = &lc
		}
		dashProfiles = append(dashProfiles, dp)
	}

	type accountKey struct {
		provider string
		account  string
	}
	accountMap := map[accountKey]*model.DashboardAccount{}
	for _, dp := range dashProfiles {
		acct := dp.Profile.Account
		if acct == "" {
			acct = "default"
		}
		k := accountKey{provider: dp.Profile.Provider, account: acct}
		if accountMap[k] == nil {
			accountMap[k] = &model.DashboardAccount{
				Provider:   dp.Profile.Provider,
				Account:    acct,
				ProfileIDs: []string{},
				Frontends:  []string{},
			}
		}
		entry := accountMap[k]
		entry.ProfileIDs = append(entry.ProfileIDs, dp.Profile.ID)
		if !slices.Contains(entry.Frontends, dp.Profile.Frontend) {
			entry.Frontends = append(entry.Frontends, dp.Profile.Frontend)
		}
		if dp.Lease != nil {
			entry.ActiveLeases++
		}
		if dp.Health != nil && dp.Health.RecentErrorRatePercent <= 3.0 {
			entry.HealthyProfiles++
		}
	}
	accounts := make([]model.DashboardAccount, 0, len(accountMap))
	for _, a := range accountMap {
		slices.Sort(a.ProfileIDs)
		slices.Sort(a.Frontends)
		accounts = append(accounts, *a)
	}
	slices.SortStableFunc(accounts, func(a, b model.DashboardAccount) int {
		if a.Provider < b.Provider {
			return -1
		}
		if a.Provider > b.Provider {
			return 1
		}
		if a.Account < b.Account {
			return -1
		}
		if a.Account > b.Account {
			return 1
		}
		return 0
	})

	policies := slices.Clone(state.Policies)
	slices.SortStableFunc(policies, func(a, b model.PolicyRule) int {
		if a.Priority > b.Priority {
			return -1
		}
		if a.Priority < b.Priority {
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return model.DashboardSummary{
		TimeUTC: now,
		Counts: map[string]int{
			"profiles":      len(state.Profiles),
			"accounts":      len(accounts),
			"policies":      len(state.Policies),
			"active_leases": len(activeLeases),
			"providers":     len(providers),
		},
		Providers:    providers,
		Profiles:     dashProfiles,
		Accounts:     accounts,
		Policies:     policies,
		ActiveLeases: activeLeases,
	}, nil
}

func (s *Service) ListLeases(ctx context.Context) ([]model.Lease, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]model.Lease, 0, len(state.Leases))
	for _, l := range state.Leases {
		if l.ExpiresAt.Before(now) {
			continue
		}
		out = append(out, l)
	}
	slices.SortStableFunc(out, func(a, b model.Lease) int {
		if a.ExpiresAt.Before(b.ExpiresAt) {
			return -1
		}
		if a.ExpiresAt.After(b.ExpiresAt) {
			return 1
		}
		return 0
	})
	return out, nil
}

func validateProfile(p model.Profile) error {
	if p.ID == "" {
		return fmt.Errorf("profile id is required")
	}
	if p.Provider == "" || p.Frontend == "" || p.AuthMethod == "" || p.Protocol == "" {
		return fmt.Errorf("provider/frontend/auth_method/protocol are required")
	}
	return nil
}
