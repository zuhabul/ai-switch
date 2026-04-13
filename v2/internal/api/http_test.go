package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/store"
	"github.com/zuhabul/ai-switch/v2/internal/vault"
)

func newTestHTTPServer(t *testing.T) (*httptest.Server, *service.Service) {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	st := store.NewFileStore(statePath)
	svc := service.New(st)
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	v := vault.NewFileVault(filepath.Join(t.TempDir(), "vault.enc.json"))
	s := NewServer(svc, v)
	return httptest.NewServer(s.Handler()), svc
}

func newTestHTTPServerWithAuth(t *testing.T, token string) (*httptest.Server, *service.Service) {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	st := store.NewFileStore(statePath)
	svc := service.New(st)
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	v := vault.NewFileVault(filepath.Join(t.TempDir(), "vault.enc.json"))
	s := NewServerWithAuth(svc, v, AuthConfig{BearerToken: token})
	return httptest.NewServer(s.Handler()), svc
}

func TestDashboardAndFrontend(t *testing.T) {
	ts, svc := newTestHTTPServer(t)
	defer ts.Close()

	ctx := context.Background()
	_ = svc.AddProfile(ctx, model.Profile{ID: "codex-main", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "codex-main", RemainingRequests5Min: 30, RemainingRequestsHour: 200, EstimatedLatencyMS: 100, RecentErrorRatePercent: 0.1})

	resp, err := http.Get(ts.URL + "/v2/dashboard/summary")
	if err != nil {
		t.Fatalf("dashboard request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	resp2, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("frontend request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for frontend, got %d", resp2.StatusCode)
	}
}

func TestRouteCandidatesAPI(t *testing.T) {
	ts, svc := newTestHTTPServer(t)
	defer ts.Close()

	ctx := context.Background()
	_ = svc.AddProfile(ctx, model.Profile{ID: "codex-main", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Priority: 10, Enabled: true})
	_ = svc.AddProfile(ctx, model.Profile{ID: "codex-backup", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Priority: 5, Enabled: true})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "codex-main", RemainingRequests5Min: 40, RemainingRequestsHour: 300, EstimatedLatencyMS: 120, RecentErrorRatePercent: 0.1})
	_ = svc.UpdateHealth(ctx, model.HealthSnapshot{ProfileID: "codex-backup", RemainingRequests5Min: 30, RemainingRequestsHour: 200, EstimatedLatencyMS: 140, RecentErrorRatePercent: 0.3})

	reqBody, _ := json.Marshal(model.TaskRequest{Frontend: "codex", TaskClass: "coding"})
	resp, err := http.Post(ts.URL+"/v2/route/candidates", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("route candidates request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var plan model.RoutePlan
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if plan.Primary.ProfileID != "codex-main" {
		t.Fatalf("expected codex-main primary, got %s", plan.Primary.ProfileID)
	}
	if len(plan.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(plan.Candidates))
	}
}

func TestAuthMiddlewareBearer(t *testing.T) {
	ts, _ := newTestHTTPServerWithAuth(t, "secret-token")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v2/dashboard/summary")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v2/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("authorized request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with bearer token, got %d", resp2.StatusCode)
	}
}

func TestIncidentsEndpoint(t *testing.T) {
	ts, svc := newTestHTTPServer(t)
	defer ts.Close()
	ctx := context.Background()
	_ = svc.AddProfile(ctx, model.Profile{ID: "codex-main", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true})
	body := `{"profile_id":"codex-main","kind":"rate_limit","message":"429","owner":"test","cooldown_seconds":120}`
	resp, err := http.Post(ts.URL+"/v2/incidents", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("incident post failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp2, err := http.Get(ts.URL + "/v2/incidents?profile_id=codex-main&limit=5")
	if err != nil {
		t.Fatalf("incident list failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d", resp2.StatusCode)
	}
}
