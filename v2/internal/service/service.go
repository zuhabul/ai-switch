package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
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

const maxIncidentLogEntries = 500

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
	p = normalizeProfile(p)
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
	p = normalizeProfile(p)
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

func (s *Service) UpsertAccount(ctx context.Context, record model.AccountRecord) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	record = normalizeAccountRecord(record)
	if err := validateAccountRecord(record); err != nil {
		return err
	}
	if state.Accounts == nil {
		state.Accounts = map[string]model.AccountRecord{}
	}
	state.Accounts[accountStorageKey(record.Provider, record.Account)] = record
	return s.store.Save(state)
}

func (s *Service) DeleteAccount(ctx context.Context, provider, account string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return err
	}
	key := accountStorageKey(provider, account)
	if _, ok := state.Accounts[key]; !ok {
		return fmt.Errorf("unknown account %s/%s", strings.TrimSpace(provider), defaultAccountLabel(strings.TrimSpace(account)))
	}
	delete(state.Accounts, key)
	return s.store.Save(state)
}

func (s *Service) ListAccountRecords(ctx context.Context) ([]model.AccountRecord, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]model.AccountRecord, 0, len(state.Accounts))
	for _, record := range state.Accounts {
		out = append(out, normalizeAccountRecord(record))
	}
	slices.SortStableFunc(out, func(a, b model.AccountRecord) int {
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
	return out, nil
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

func (s *Service) RecordIncident(ctx context.Context, in model.Incident) (model.Incident, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load()
	if err != nil {
		return model.Incident{}, err
	}
	if in.ProfileID == "" {
		return model.Incident{}, fmt.Errorf("profile_id is required")
	}
	p, ok := state.Profiles[in.ProfileID]
	if !ok {
		return model.Incident{}, fmt.Errorf("unknown profile %s", in.ProfileID)
	}
	if in.ID == "" {
		in.ID = fmt.Sprintf("inc_%d", time.Now().UTC().UnixNano())
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now().UTC()
	}
	if in.Kind == "" {
		in.Kind = "generic"
	}
	if in.CooldownSeconds > 0 {
		p.CooldownUntil = time.Now().UTC().Add(time.Duration(in.CooldownSeconds) * time.Second)
		state.Profiles[p.ID] = p
	}
	if strings.EqualFold(in.Kind, "rate_limit") {
		h := state.Health[in.ProfileID]
		h.ProfileID = in.ProfileID
		h.UpdatedAt = time.Now().UTC()
		if h.RemainingRequests5Min > 0 {
			h.RemainingRequests5Min = 0
		}
		if h.RemainingRequestsHour > 0 {
			h.RemainingRequestsHour = max(0, h.RemainingRequestsHour/2)
		}
		if h.RecentErrorRatePercent < 5 {
			h.RecentErrorRatePercent = 5
		}
		state.Health[in.ProfileID] = h
	}
	state.Incidents = append(state.Incidents, in)
	if len(state.Incidents) > maxIncidentLogEntries {
		state.Incidents = slices.Clone(state.Incidents[len(state.Incidents)-maxIncidentLogEntries:])
	}
	if err := s.store.Save(state); err != nil {
		return model.Incident{}, err
	}
	return in, nil
}

func (s *Service) ListIncidents(ctx context.Context, profileID string, limit int) ([]model.Incident, error) {
	_ = ctx
	state, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	out := make([]model.Incident, 0, len(state.Incidents))
	for i := len(state.Incidents) - 1; i >= 0; i-- {
		it := state.Incidents[i]
		if profileID != "" && it.ProfileID != profileID {
			continue
		}
		out = append(out, it)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
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
	p, ok := state.Profiles[profileID]
	if !ok {
		return model.Lease{}, fmt.Errorf("unknown profile %s", profileID)
	}
	if !ownerAllowed(p.OwnerScopes, owner) {
		return model.Lease{}, fmt.Errorf("owner %q is not allowed for profile %s", owner, profileID)
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
	healthScoreTotal := map[accountKey]float64{}
	healthScoreCount := map[accountKey]int{}
	withRecordMeta := map[accountKey]bool{}
	ensureAccount := func(provider, account string) (*model.DashboardAccount, accountKey) {
		k := accountKey{
			provider: strings.TrimSpace(provider),
			account:  defaultAccountLabel(strings.TrimSpace(account)),
		}
		if accountMap[k] == nil {
			accountMap[k] = &model.DashboardAccount{
				Provider:   k.provider,
				Account:    k.account,
				ProfileIDs: []string{},
				Frontends:  []string{},
				Tags:       []string{},
			}
		}
		return accountMap[k], k
	}
	for _, dp := range dashProfiles {
		entry, k := ensureAccount(dp.Profile.Provider, dp.Profile.Account)
		entry.ProfileIDs = append(entry.ProfileIDs, dp.Profile.ID)
		if !slices.Contains(entry.Frontends, dp.Profile.Frontend) {
			entry.Frontends = append(entry.Frontends, dp.Profile.Frontend)
		}
		if dp.Lease != nil {
			entry.ActiveLeases++
		}
		if dp.Profile.CooldownUntil.After(now) {
			entry.CooldownProfiles++
		}
		if dp.Health != nil && dp.Health.RecentErrorRatePercent <= 3.0 {
			entry.HealthyProfiles++
		}
		if dp.Health != nil {
			score := accountHealthScore(*dp.Health)
			healthScoreTotal[k] += score
			healthScoreCount[k]++
		}
	}

	for _, record := range state.Accounts {
		record = normalizeAccountRecord(record)
		if _, ok := providers[record.Provider]; !ok {
			providers[record.Provider] = 0
		}
		entry, k := ensureAccount(record.Provider, record.Account)
		withRecordMeta[k] = true
		entry.Status = record.Status
		entry.Tier = record.Tier
		entry.AuthMethod = record.AuthMethod
		entry.DailyLimitUSD = record.DailyLimitUSD
		entry.DailyUsedUSD = record.DailyUsedUSD
		entry.DailyRemainingUSD = remainingBudget(record.DailyLimitUSD, record.DailyUsedUSD)
		entry.DailyUsagePercent = usagePercent(record.DailyUsedUSD, record.DailyLimitUSD)
		entry.MonthlyLimitUSD = record.MonthlyLimitUSD
		entry.MonthlyUsedUSD = record.MonthlyUsedUSD
		entry.MonthlyRemainingUSD = remainingBudget(record.MonthlyLimitUSD, record.MonthlyUsedUSD)
		entry.MonthlyUsagePercent = usagePercent(record.MonthlyUsedUSD, record.MonthlyLimitUSD)
		entry.RateLimitRemaining5Min = record.RateLimitRemaining5Min
		entry.RateLimitRemainingHour = record.RateLimitRemainingHour
		entry.Tags = slices.Clone(record.Tags)
		entry.Notes = record.Notes

		if !record.AuthExpiresAt.IsZero() {
			t := record.AuthExpiresAt
			entry.AuthExpiresAt = &t
		}
		if !record.DailyResetAt.IsZero() {
			t := record.DailyResetAt
			entry.DailyResetAt = &t
		}
		if !record.MonthlyResetAt.IsZero() {
			t := record.MonthlyResetAt
			entry.MonthlyResetAt = &t
		}
		if !record.RateLimitResetAt.IsZero() {
			t := record.RateLimitResetAt
			entry.RateLimitResetAt = &t
		}
		if !record.LastCheckedAt.IsZero() {
			t := record.LastCheckedAt
			entry.LastCheckedAt = &t
		}
	}
	accounts := make([]model.DashboardAccount, 0, len(accountMap))
	for k, a := range accountMap {
		slices.Sort(a.ProfileIDs)
		slices.Sort(a.Frontends)
		slices.Sort(a.Tags)
		a.ProfileCount = len(a.ProfileIDs)
		if healthScoreCount[k] > 0 {
			a.HealthScore = round2(healthScoreTotal[k] / float64(healthScoreCount[k]))
		} else if a.ProfileCount > 0 {
			a.HealthScore = round2((float64(a.HealthyProfiles) / float64(a.ProfileCount)) * 100)
		}
		if !withRecordMeta[k] || strings.TrimSpace(a.Status) == "" {
			a.Status = inferAccountStatus(*a)
		}
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

	recentIncidents := make([]model.Incident, 0, 20)
	for i := len(state.Incidents) - 1; i >= 0 && len(recentIncidents) < 20; i-- {
		recentIncidents = append(recentIncidents, state.Incidents[i])
	}

	return model.DashboardSummary{
		TimeUTC: now,
		Counts: map[string]int{
			"profiles":      len(state.Profiles),
			"accounts":      len(accounts),
			"policies":      len(state.Policies),
			"active_leases": len(activeLeases),
			"providers":     len(providers),
			"incidents":     len(state.Incidents),
		},
		Providers:       providers,
		Profiles:        dashProfiles,
		Accounts:        accounts,
		Policies:        policies,
		ActiveLeases:    activeLeases,
		RecentIncidents: recentIncidents,
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

func ownerAllowed(ownerScopes []string, owner string) bool {
	if len(ownerScopes) == 0 {
		return true
	}
	if owner == "" {
		return false
	}
	for _, s := range ownerScopes {
		if s == "*" || s == owner {
			return true
		}
	}
	return false
}

func normalizeProfile(p model.Profile) model.Profile {
	p.ID = strings.TrimSpace(p.ID)
	p.Provider = strings.TrimSpace(p.Provider)
	p.Frontend = strings.TrimSpace(p.Frontend)
	p.AuthMethod = strings.TrimSpace(p.AuthMethod)
	p.Protocol = strings.TrimSpace(p.Protocol)
	p.Account = strings.TrimSpace(p.Account)
	scopes := make([]string, 0, len(p.OwnerScopes))
	seen := map[string]struct{}{}
	for _, s := range p.OwnerScopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		scopes = append(scopes, s)
	}
	p.OwnerScopes = scopes
	return p
}

func accountStorageKey(provider, account string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "::" + strings.ToLower(defaultAccountLabel(strings.TrimSpace(account)))
}

func defaultAccountLabel(account string) string {
	if strings.TrimSpace(account) == "" {
		return "default"
	}
	return strings.TrimSpace(account)
}

func validateAccountRecord(record model.AccountRecord) error {
	if strings.TrimSpace(record.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(record.Account) == "" {
		return fmt.Errorf("account is required")
	}
	if record.DailyLimitUSD < 0 || record.DailyUsedUSD < 0 || record.MonthlyLimitUSD < 0 || record.MonthlyUsedUSD < 0 {
		return fmt.Errorf("limit and usage values cannot be negative")
	}
	return nil
}

func normalizeAccountRecord(record model.AccountRecord) model.AccountRecord {
	record.Provider = strings.TrimSpace(record.Provider)
	record.Account = defaultAccountLabel(record.Account)
	record.Status = strings.TrimSpace(strings.ToLower(record.Status))
	record.Tier = strings.TrimSpace(record.Tier)
	record.AuthMethod = strings.TrimSpace(record.AuthMethod)
	record.Notes = strings.TrimSpace(record.Notes)
	if record.LastCheckedAt.IsZero() {
		record.LastCheckedAt = time.Now().UTC()
	}
	tags := make([]string, 0, len(record.Tags))
	seen := map[string]struct{}{}
	for _, tag := range record.Tags {
		clean := strings.TrimSpace(strings.ToLower(tag))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		tags = append(tags, clean)
	}
	record.Tags = tags
	return record
}

func inferAccountStatus(account model.DashboardAccount) string {
	if account.ProfileCount == 0 {
		return "standby"
	}
	if account.CooldownProfiles >= account.ProfileCount {
		return "cooldown"
	}
	if account.HealthyProfiles == 0 {
		return "degraded"
	}
	return "healthy"
}

func remainingBudget(limit, used float64) float64 {
	if limit <= 0 {
		return 0
	}
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return round2(remaining)
}

func usagePercent(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	return round2(clamp((used/limit)*100, 0, 999))
}

func accountHealthScore(h model.HealthSnapshot) float64 {
	score := 100.0
	score -= clamp(h.RecentErrorRatePercent*18.0, 0, 75)
	score -= clamp(float64(h.EstimatedLatencyMS)/20.0, 0, 22)
	score += clamp(float64(h.RemainingRequests5Min)/6.0, 0, 8)
	score += clamp(float64(h.RemainingRequestsHour)/120.0, 0, 8)
	return round2(clamp(score, 0, 100))
}

func round2(v float64) float64 {
	if v >= 0 {
		return float64(int(v*100+0.5)) / 100
	}
	return float64(int(v*100-0.5)) / 100
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
