package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/service"
)

type Server struct {
	svc *service.Service
}

func NewServer(svc *service.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v2/profiles", s.profiles)
	mux.HandleFunc("/v2/policies", s.policies)
	mux.HandleFunc("/v2/leases", s.leases)
	mux.HandleFunc("/v2/route", s.route)
	mux.HandleFunc("/v2/health", s.healthUpdate)
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
