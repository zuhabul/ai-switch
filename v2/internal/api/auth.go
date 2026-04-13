package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AuthConfig struct {
	BearerToken string
	HMACKeys    map[string]string
}

func (c AuthConfig) Enabled() bool {
	return strings.TrimSpace(c.BearerToken) != "" || len(c.HMACKeys) > 0
}

func (c AuthConfig) enforce(next http.Handler) http.Handler {
	if !c.Enabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if c.validBearer(r) {
			next.ServeHTTP(w, r)
			return
		}
		if ok, err := c.validHMAC(r); ok {
			next.ServeHTTP(w, r)
			return
		} else if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
	})
}

func requiresAuth(path string) bool {
	return strings.HasPrefix(path, "/v2/") || path == "/metrics"
}

func (c AuthConfig) validBearer(r *http.Request) bool {
	token := strings.TrimSpace(c.BearerToken)
	if token == "" {
		return false
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix)) == token
}

func (c AuthConfig) validHMAC(r *http.Request) (bool, error) {
	if len(c.HMACKeys) == 0 {
		return false, nil
	}
	keyID := strings.TrimSpace(r.Header.Get("X-AISWITCH-Key-ID"))
	tsRaw := strings.TrimSpace(r.Header.Get("X-AISWITCH-Timestamp"))
	sig := strings.TrimSpace(r.Header.Get("X-AISWITCH-Signature"))
	if keyID == "" || tsRaw == "" || sig == "" {
		return false, nil
	}
	secret := c.HMACKeys[keyID]
	if secret == "" {
		return false, fmt.Errorf("unknown key id")
	}
	ts, err := parseTimestamp(tsRaw)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp")
	}
	if skew := time.Since(ts); skew > 5*time.Minute || skew < -5*time.Minute {
		return false, fmt.Errorf("timestamp out of range")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("cannot read body")
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	bodySum := sha256.Sum256(body)
	payload := strings.Join([]string{
		strings.ToUpper(r.Method),
		r.URL.Path,
		tsRaw,
		hex.EncodeToString(bodySum[:]),
	}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(strings.ToLower(expected))) {
		return false, fmt.Errorf("invalid signature")
	}
	return true, nil
}

func parseTimestamp(v string) (time.Time, error) {
	if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC(), nil
	}
	return time.Parse(time.RFC3339, v)
}
