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
	if d.ProfileID != "b" {
		t.Fatalf("expected b, got %s", d.ProfileID)
	}
}
