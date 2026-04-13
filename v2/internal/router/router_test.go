package router

import (
	"testing"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

func TestPickBest(t *testing.T) {
	now := time.Now().UTC()
	decision := PickBest(Input{
		Now: now,
		Profiles: []model.Profile{
			{ID: "a", Provider: "openai", Frontend: "codex", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, Priority: 1},
			{ID: "b", Provider: "google", Frontend: "gemini_cli", Protocol: "native_cli", AuthMethod: "api_key", Enabled: true, Priority: 4},
		},
		Health: map[string]model.HealthSnapshot{
			"a": {RemainingRequests5Min: 15, RemainingRequestsHour: 100, EstimatedLatencyMS: 300, RecentErrorRatePercent: 1},
			"b": {RemainingRequests5Min: 45, RemainingRequestsHour: 500, EstimatedLatencyMS: 90, RecentErrorRatePercent: 0.1},
		},
		Request: model.TaskRequest{Frontend: "codex"},
	})
	if decision.ProfileID != "b" {
		t.Fatalf("expected b, got %s, reasons=%v rejected=%v", decision.ProfileID, decision.Reasons, decision.Rejected)
	}
}

func TestCooldownRejected(t *testing.T) {
	now := time.Now().UTC()
	d := PickBest(Input{
		Now: now,
		Profiles: []model.Profile{
			{ID: "a", Provider: "openai", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, CooldownUntil: now.Add(10 * time.Minute)},
		},
		Health:  map[string]model.HealthSnapshot{"a": {RemainingRequests5Min: 99}},
		Request: model.TaskRequest{},
	})
	if d.ProfileID != "" {
		t.Fatalf("expected none, got %s", d.ProfileID)
	}
	if len(d.Rejected) == 0 {
		t.Fatalf("expected rejection reasons")
	}
}
