package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	st := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	svc := New(st)
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return svc
}

func TestLeaseLock(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.AddProfile(ctx, model.Profile{ID: "p1", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true}); err != nil {
		t.Fatalf("add profile: %v", err)
	}
	if _, err := svc.AcquireLease(ctx, "p1", "codex", "a", 10*time.Minute); err != nil {
		t.Fatalf("acquire 1 failed: %v", err)
	}
	if _, err := svc.AcquireLease(ctx, "p1", "codex", "b", 10*time.Minute); err == nil {
		t.Fatalf("expected second lease to fail")
	}
}

func TestRoute(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	_ = svc.AddProfile(ctx, model.Profile{ID: "a", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true, Priority: 1})
	_ = svc.AddProfile(ctx, model.Profile{ID: "b", Provider: "google", Frontend: "gemini_cli", AuthMethod: "api_key", Protocol: "native_cli", Enabled: true, Priority: 3})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "a", RemainingRequests5Min: 10, RemainingRequestsHour: 20, EstimatedLatencyMS: 320, RecentErrorRatePercent: 2})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "b", RemainingRequests5Min: 30, RemainingRequestsHour: 200, EstimatedLatencyMS: 80, RecentErrorRatePercent: 0.1})

	d, err := svc.Route(ctx, model.TaskRequest{Frontend: "codex", TaskClass: "coding"})
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if d.ProfileID != "a" {
		t.Fatalf("expected a (frontend-scoped), got %s", d.ProfileID)
	}
}

func TestLeaseReacquireSameOwnerRefreshes(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.AddProfile(ctx, model.Profile{ID: "p1", Provider: "google", Frontend: "gemini_cli", AuthMethod: "google_login", Protocol: "native_cli", Enabled: true}); err != nil {
		t.Fatalf("add profile: %v", err)
	}

	first, err := svc.AcquireLease(ctx, "p1", "gemini_cli", "wrapper-echo", 5*time.Minute)
	if err != nil {
		t.Fatalf("acquire first lease failed: %v", err)
	}
	second, err := svc.AcquireLease(ctx, "p1", "gemini_cli", "wrapper-echo", 10*time.Minute)
	if err != nil {
		t.Fatalf("acquire same-owner lease failed: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same lease id to be refreshed, got %s then %s", first.ID, second.ID)
	}
	if !second.ExpiresAt.After(first.ExpiresAt) {
		t.Fatalf("expected refreshed lease expiry to extend: first=%s second=%s", first.ExpiresAt, second.ExpiresAt)
	}
}

func TestSecretBindings(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.AddProfile(ctx, model.Profile{ID: "p1", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true}); err != nil {
		t.Fatalf("add profile: %v", err)
	}
	if err := svc.BindSecret(ctx, "p1", "OPENAI_API_KEY", "openai-main"); err != nil {
		t.Fatalf("bind: %v", err)
	}
	bindings, err := svc.ListSecretBindings(ctx, "p1")
	if err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if bindings["OPENAI_API_KEY"] != "openai-main" {
		t.Fatalf("binding mismatch: %v", bindings)
	}
	if err := svc.UnbindSecret(ctx, "p1", "OPENAI_API_KEY"); err != nil {
		t.Fatalf("unbind: %v", err)
	}
}

func TestProfileCRUDAndRoutePlan(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.AddProfile(ctx, model.Profile{
		ID:         "codex-main",
		Provider:   "openai",
		Frontend:   "codex",
		AuthMethod: "chatgpt",
		Protocol:   "app_server",
		Account:    "openai-a",
		Priority:   10,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add codex-main: %v", err)
	}
	if err := svc.AddProfile(ctx, model.Profile{
		ID:         "codex-backup",
		Provider:   "openai",
		Frontend:   "codex",
		AuthMethod: "chatgpt",
		Protocol:   "app_server",
		Account:    "openai-b",
		Priority:   5,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add codex-backup: %v", err)
	}
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "codex-main", RemainingRequests5Min: 30, RemainingRequestsHour: 200, EstimatedLatencyMS: 130, RecentErrorRatePercent: 0.2})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "codex-backup", RemainingRequests5Min: 20, RemainingRequestsHour: 140, EstimatedLatencyMS: 180, RecentErrorRatePercent: 0.6})

	plan, err := svc.RoutePlan(ctx, model.TaskRequest{Frontend: "codex", TaskClass: "coding"})
	if err != nil {
		t.Fatalf("route plan failed: %v", err)
	}
	if plan.Primary.ProfileID != "codex-main" {
		t.Fatalf("expected codex-main primary, got %s", plan.Primary.ProfileID)
	}
	if len(plan.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(plan.Candidates))
	}

	updated := model.Profile{
		ID:         "codex-backup",
		Provider:   "openai",
		Frontend:   "codex",
		AuthMethod: "chatgpt",
		Protocol:   "app_server",
		Account:    "openai-b",
		Priority:   12,
		Enabled:    true,
	}
	if err := svc.UpdateProfile(ctx, updated); err != nil {
		t.Fatalf("update profile failed: %v", err)
	}
	got, err := svc.GetProfile(ctx, "codex-backup")
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if got.Priority != 12 {
		t.Fatalf("expected updated priority=12, got %d", got.Priority)
	}
	if err := svc.DeleteProfile(ctx, "codex-backup"); err != nil {
		t.Fatalf("delete profile failed: %v", err)
	}
	if _, err := svc.GetProfile(ctx, "codex-backup"); err == nil {
		t.Fatalf("expected deleted profile lookup to fail")
	}
}

func TestPolicyUpdateDelete(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	rule := model.PolicyRule{Name: "prod-only", Priority: 100, Frontends: []string{"codex"}}
	if err := svc.AddPolicy(ctx, rule); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}
	rule.Priority = 120
	rule.RequireAnyTag = []string{"prod"}
	if err := svc.UpdatePolicy(ctx, rule); err != nil {
		t.Fatalf("update policy failed: %v", err)
	}
	got, err := svc.GetPolicy(ctx, "prod-only")
	if err != nil {
		t.Fatalf("get policy failed: %v", err)
	}
	if got.Priority != 120 {
		t.Fatalf("expected updated priority 120, got %d", got.Priority)
	}
	if err := svc.DeletePolicy(ctx, "prod-only"); err != nil {
		t.Fatalf("delete policy failed: %v", err)
	}
	if _, err := svc.GetPolicy(ctx, "prod-only"); err == nil {
		t.Fatalf("expected deleted policy lookup to fail")
	}
}

func TestDashboardSummary(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	_ = svc.AddProfile(ctx, model.Profile{ID: "p1", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Account: "a1", Enabled: true})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "p1", RemainingRequests5Min: 40, RemainingRequestsHour: 300, EstimatedLatencyMS: 120, RecentErrorRatePercent: 0.1})
	_ = svc.BindSecret(ctx, "p1", "OPENAI_API_KEY", "key-a")
	_, _ = svc.AcquireLease(ctx, "p1", "codex", "test-owner", 5*time.Minute)
	summary, err := svc.DashboardSummary(ctx)
	if err != nil {
		t.Fatalf("dashboard summary failed: %v", err)
	}
	if summary.Counts["profiles"] != 1 {
		t.Fatalf("expected profiles count 1, got %d", summary.Counts["profiles"])
	}
	if summary.Counts["active_leases"] != 1 {
		t.Fatalf("expected active leases 1, got %d", summary.Counts["active_leases"])
	}
	if len(summary.Profiles) != 1 || summary.Profiles[0].SecretCount != 1 {
		t.Fatalf("expected 1 profile with secret count 1, got %+v", summary.Profiles)
	}
	if len(summary.Accounts) != 1 {
		t.Fatalf("expected 1 account aggregation, got %d", len(summary.Accounts))
	}
}
