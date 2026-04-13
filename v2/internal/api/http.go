package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/adapter"
	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/vault"
)

type Server struct {
	svc   *service.Service
	vault *vault.FileVault
}

func NewServer(svc *service.Service, v *vault.FileVault) *Server {
	return &Server{svc: svc, vault: v}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v2/profiles", s.profiles)
	mux.HandleFunc("/v2/policies", s.policies)
	mux.HandleFunc("/v2/leases", s.leases)
	mux.HandleFunc("/v2/route", s.route)
	mux.HandleFunc("/v2/health", s.healthUpdate)
	mux.HandleFunc("/v2/secret-bindings", s.secretBindings)
	mux.HandleFunc("/v2/secrets", s.secrets)
	mux.HandleFunc("/v2/runtime/plan", s.runtimePlan)
	return loggingMiddleware(mux)
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC()})
}

func (s *Server) profiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) policies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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

func (s *Server) healthUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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
	case http.MethodPost:
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
	case http.MethodPost:
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
	decision, err := s.svc.Route(r.Context(), model.TaskRequest{
		Frontend:           req.Frontend,
		TaskClass:          req.TaskClass,
		RequiredProtocol:   req.RequiredProtocol,
		PreferredProviders: req.PreferredProviders,
		RequireTags:        req.RequireTags,
		Owner:              req.Owner,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "decision": decision})
		return
	}

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
