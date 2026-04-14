package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/adapter"
	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/vault"
)

//go:embed web/index.html web/app.js web/styles.css
var webFiles embed.FS

type Server struct {
	svc      *service.Service
	vault    *vault.FileVault
	adapters *adapter.Registry
	hooks    *adapter.HookRegistry
	auth     AuthConfig
}

func NewServer(svc *service.Service, v *vault.FileVault) *Server {
	return NewServerWithAuth(svc, v, AuthConfig{})
}

func NewServerWithAuth(svc *service.Service, v *vault.FileVault, auth AuthConfig) *Server {
	return &Server{
		svc:      svc,
		vault:    v,
		adapters: adapter.NewRegistry(),
		hooks:    adapter.NewHookRegistry(),
		auth:     auth,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v2/profiles", s.profiles)
	mux.HandleFunc("/v2/policies", s.policies)
	mux.HandleFunc("/v2/leases", s.leases)
	mux.HandleFunc("/v2/route", s.route)
	mux.HandleFunc("/v2/route/candidates", s.routeCandidates)
	mux.HandleFunc("/v2/health", s.healthUpdate)
	mux.HandleFunc("/v2/secret-bindings", s.secretBindings)
	mux.HandleFunc("/v2/secrets", s.secrets)
	mux.HandleFunc("/v2/runtime/plan", s.runtimePlan)
	mux.HandleFunc("/v2/dashboard/summary", s.dashboardSummary)
	mux.HandleFunc("/v2/accounts", s.accounts)
	mux.HandleFunc("/v2/accounts/failover", s.accountFailover)
	mux.HandleFunc("/v2/adapters", s.adaptersInfo)
	mux.HandleFunc("/v2/adapters/contract", s.adaptersContract)
	mux.HandleFunc("/v2/incidents", s.incidents)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/", s.frontend)
	return loggingMiddleware(s.auth.enforce(mux))
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC()})
}

func (s *Server) profiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id != "" {
			p, err := s.svc.GetProfile(r.Context(), id)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, p)
			return
		}
		ps, err := s.svc.ListProfiles(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ps)
	case http.MethodPost:
		var p model.Profile
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.AddProfile(r.Context(), p); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	case http.MethodPut:
		var p model.Profile
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.UpdateProfile(r.Context(), p); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
			return
		}
		if err := s.svc.DeleteProfile(r.Context(), id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) policies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name != "" {
			policy, err := s.svc.GetPolicy(r.Context(), name)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, policy)
			return
		}
		policies, err := s.svc.ListPolicies(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, policies)
	case http.MethodPost:
		var p model.PolicyRule
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.AddPolicy(r.Context(), p); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	case http.MethodPut:
		var p model.PolicyRule
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.UpdatePolicy(r.Context(), p); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case http.MethodDelete:
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		if err := s.svc.DeletePolicy(r.Context(), name); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) leases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		leases, err := s.svc.ListLeases(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, leases)
	case http.MethodPost:
		var req struct {
			ProfileID string `json:"profile_id"`
			Frontend  string `json:"frontend"`
			Owner     string `json:"owner"`
			TTLMin    int    `json:"ttl_min"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err)
			return
		}
		lease, err := s.svc.AcquireLease(r.Context(), req.ProfileID, req.Frontend, req.Owner, time.Duration(req.TTLMin)*time.Minute)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, lease)
	case http.MethodDelete:
		leaseID := r.URL.Query().Get("lease_id")
		if leaseID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "lease_id is required"})
			return
		}
		if err := s.svc.ReleaseLease(r.Context(), leaseID); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req model.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	decision, err := s.svc.Route(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "decision": decision})
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func (s *Server) routeCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req model.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	plan, err := s.svc.RoutePlan(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "route_plan": plan})
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) healthUpdate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		health, err := s.svc.ListHealth(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, health)
	case http.MethodPost:
		var hs model.HealthSnapshot
		if err := json.NewDecoder(r.Body).Decode(&hs); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.UpdateHealth(r.Context(), hs); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) incidents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profileID := strings.TrimSpace(r.URL.Query().Get("profile_id"))
		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				limit = n
			}
		}
		items, err := s.svc.ListIncidents(r.Context(), profileID, limit)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req struct {
			ProfileID       string `json:"profile_id"`
			Kind            string `json:"kind"`
			Message         string `json:"message"`
			Owner           string `json:"owner"`
			CooldownSeconds int    `json:"cooldown_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err)
			return
		}
		incident, err := s.svc.RecordIncident(r.Context(), model.Incident{
			ProfileID:       req.ProfileID,
			Kind:            req.Kind,
			Message:         req.Message,
			Owner:           req.Owner,
			CooldownSeconds: req.CooldownSeconds,
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, incident)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) secretBindings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profileID := r.URL.Query().Get("profile_id")
		if profileID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "profile_id is required"})
			return
		}
		bindings, err := s.svc.ListSecretBindings(r.Context(), profileID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, bindings)
	case http.MethodPost, http.MethodPut:
		var req struct {
			ProfileID string `json:"profile_id"`
			EnvVar    string `json:"env_var"`
			SecretKey string `json:"secret_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.BindSecret(r.Context(), req.ProfileID, req.EnvVar, req.SecretKey); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	case http.MethodDelete:
		profileID := r.URL.Query().Get("profile_id")
		envVar := r.URL.Query().Get("env_var")
		if profileID == "" || envVar == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "profile_id and env_var are required"})
			return
		}
		if err := s.svc.UnbindSecret(r.Context(), profileID, envVar); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) secrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		names, err := s.vault.List()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": names})
	case http.MethodPost, http.MethodPut:
		var req struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.vault.Set(req.Name, req.Value); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		if err := s.vault.Delete(name); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) runtimePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req model.RuntimePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	if req.Frontend == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "frontend is required"})
		return
	}
	if req.TaskClass == "" {
		req.TaskClass = "coding"
	}
	if req.Owner == "" {
		req.Owner = "runtime-plan-api"
	}
	routePlan, err := s.svc.RoutePlan(r.Context(), model.TaskRequest{
		Frontend:           req.Frontend,
		TaskClass:          req.TaskClass,
		RequiredProtocol:   req.RequiredProtocol,
		PreferredProviders: req.PreferredProviders,
		RequireTags:        req.RequireTags,
		Owner:              req.Owner,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "route_plan": routePlan})
		return
	}
	decision := routePlan.Primary

	ttl := 15 * time.Minute
	if req.LeaseTTLSeconds > 0 {
		ttl = time.Duration(req.LeaseTTLSeconds) * time.Second
	}
	lease, err := s.svc.AcquireLease(r.Context(), decision.ProfileID, req.Frontend, req.Owner, ttl)
	if err != nil {
		writeErr(w, err)
		return
	}
	releaseOnErr := true
	defer func() {
		if releaseOnErr {
			_ = s.svc.ReleaseLease(r.Context(), lease.ID)
		}
	}()

	profile, err := s.svc.GetProfile(r.Context(), decision.ProfileID)
	if err != nil {
		writeErr(w, err)
		return
	}

	bindings, err := s.svc.ListSecretBindings(r.Context(), decision.ProfileID)
	if err != nil {
		writeErr(w, err)
		return
	}
	env := map[string]string{
		"AI_SWITCH_PROFILE_ID": decision.ProfileID,
		"AI_SWITCH_LEASE_ID":   lease.ID,
	}
	fallbackIDs := make([]string, 0, len(routePlan.Candidates))
	for _, c := range routePlan.Candidates {
		if c.ProfileID == "" || c.ProfileID == decision.ProfileID {
			continue
		}
		fallbackIDs = append(fallbackIDs, c.ProfileID)
	}
	if len(fallbackIDs) > 0 {
		env["AI_SWITCH_FAILOVER_PROFILE_IDS"] = strings.Join(fallbackIDs, ",")
	}
	for envVar, secretKey := range bindings {
		value, err := s.vault.Get(secretKey)
		if err != nil {
			writeErr(w, err)
			return
		}
		env[envVar] = value
	}

	spec, err := adapter.BuildDefault(profile.Frontend, adapter.LaunchRequest{
		Frontend: req.Frontend,
		Prompt:   req.Prompt,
		Cwd:      req.Cwd,
		Model:    req.Model,
		Args:     req.CommandArgs,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	for k, v := range spec.Env {
		env[k] = v
	}

	releaseOnErr = false
	writeJSON(w, http.StatusOK, model.RuntimePlan{
		ProfileID: decision.ProfileID,
		LeaseID:   lease.ID,
		Command:   spec.Command,
		Args:      spec.Args,
		Env:       env,
		Reasons:   decision.Reasons,
	})
}

func (s *Server) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	summary, err := s.svc.DashboardSummary(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		summary, err := s.svc.DashboardSummary(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		provider := strings.TrimSpace(r.URL.Query().Get("provider"))
		account := strings.TrimSpace(r.URL.Query().Get("account"))
		items := make([]model.DashboardAccount, 0, len(summary.Accounts))
		for _, item := range summary.Accounts {
			if provider != "" && !strings.EqualFold(item.Provider, provider) {
				continue
			}
			if account != "" && !strings.EqualFold(item.Account, account) {
				continue
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPut:
		var record model.AccountRecord
		if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
			writeErr(w, err)
			return
		}
		if err := s.svc.UpsertAccount(r.Context(), record); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	case http.MethodDelete:
		provider := strings.TrimSpace(r.URL.Query().Get("provider"))
		account := strings.TrimSpace(r.URL.Query().Get("account"))
		if provider == "" || account == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "provider and account are required"})
			return
		}
		if err := s.svc.DeleteAccount(r.Context(), provider, account); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) accountFailover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Provider        string `json:"provider"`
		Account         string `json:"account"`
		Owner           string `json:"owner"`
		Message         string `json:"message"`
		CooldownSeconds int    `json:"cooldown_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	if strings.TrimSpace(req.Owner) == "" {
		req.Owner = "dashboard"
	}
	result, err := s.svc.TriggerAccountFailover(
		r.Context(),
		req.Provider,
		req.Account,
		req.Owner,
		req.Message,
		time.Duration(req.CooldownSeconds)*time.Second,
	)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) adaptersInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities":      s.adapters.List(),
		"runtime_frontends": s.hooks.ListFrontends(),
	})
}

func (s *Server) adaptersContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, adapter.DefaultContract())
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	summary, err := s.svc.DashboardSummary(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	lines := []string{
		"# HELP aiswitch_profiles_total Number of configured profiles.",
		"# TYPE aiswitch_profiles_total gauge",
		fmt.Sprintf("aiswitch_profiles_total %d", summary.Counts["profiles"]),
		"# HELP aiswitch_accounts_total Number of provider account groups.",
		"# TYPE aiswitch_accounts_total gauge",
		fmt.Sprintf("aiswitch_accounts_total %d", summary.Counts["accounts"]),
		"# HELP aiswitch_policies_total Number of active policy rules.",
		"# TYPE aiswitch_policies_total gauge",
		fmt.Sprintf("aiswitch_policies_total %d", summary.Counts["policies"]),
		"# HELP aiswitch_active_leases_total Number of active leases.",
		"# TYPE aiswitch_active_leases_total gauge",
		fmt.Sprintf("aiswitch_active_leases_total %d", summary.Counts["active_leases"]),
		"# HELP aiswitch_incidents_total Number of tracked incidents.",
		"# TYPE aiswitch_incidents_total gauge",
		fmt.Sprintf("aiswitch_incidents_total %d", summary.Counts["incidents"]),
	}
	for provider, count := range summary.Providers {
		lines = append(lines, fmt.Sprintf(`aiswitch_provider_profiles_total{provider=%q} %d`, provider, count))
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}

func (s *Server) frontend(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		serveEmbeddedFile(w, "web/index.html", "text/html; charset=utf-8")
	case "/app.js":
		serveEmbeddedFile(w, "web/app.js", "application/javascript; charset=utf-8")
	case "/styles.css":
		serveEmbeddedFile(w, "web/styles.css", "text/css; charset=utf-8")
	case "/favicon.ico":
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func serveEmbeddedFile(w http.ResponseWriter, name, contentType string) {
	b, err := fs.ReadFile(webFiles, name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration_ms=%d", r.Method, r.URL.Path, time.Since(start).Milliseconds())
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
}
