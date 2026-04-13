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
	if p.ID == "" {
		return fmt.Errorf("profile id is required")
	}
	if p.Provider == "" || p.Frontend == "" || p.AuthMethod == "" || p.Protocol == "" {
		return fmt.Errorf("provider/frontend/auth_method/protocol are required")
	}
	if !p.Enabled {
		p.Enabled = true
	}
	state.Profiles[p.ID] = p
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
	for _, l := range state.Leases {
		if l.ProfileID == profileID && l.ExpiresAt.After(now) {
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
