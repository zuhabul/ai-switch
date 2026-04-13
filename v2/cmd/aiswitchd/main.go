package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/api"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/store"
	"github.com/zuhabul/ai-switch/v2/internal/vault"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4417", "listen address")
	statePath := flag.String("state", defaultStatePath(), "state file path")
	vaultPath := flag.String("vault", defaultVaultPath(), "vault file path")
	apiToken := flag.String("api-token", os.Getenv("AISWITCHD_API_TOKEN"), "optional bearer API token for /v2 and /metrics")
	hmacKeysRaw := flag.String("hmac-keys", os.Getenv("AISWITCHD_HMAC_KEYS"), "optional HMAC keys: key1:secret1,key2:secret2")
	flag.Parse()

	st := store.NewFileStore(*statePath)
	svc := service.New(st)
	v := vault.NewFileVault(*vaultPath)
	if err := svc.Init(context.Background()); err != nil {
		log.Fatalf("init failed: %v", err)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           api.NewServerWithAuth(svc, v, api.AuthConfig{BearerToken: strings.TrimSpace(*apiToken), HMACKeys: parseHMACKeys(*hmacKeysRaw)}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("aiswitchd listening on %s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}
}

func defaultStatePath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ".aiswitch/state.json"
	}
	return filepath.Join(h, ".config", "ai-switch-v2", "state.json")
}

func defaultVaultPath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ".aiswitch/secrets.enc.json"
	}
	return filepath.Join(h, ".config", "ai-switch-v2", "secrets.enc.json")
}

func parseHMACKeys(raw string) map[string]string {
	out := map[string]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		keyID := strings.TrimSpace(parts[0])
		secret := strings.TrimSpace(parts[1])
		if keyID == "" || secret == "" {
			continue
		}
		out[keyID] = secret
	}
	return out
}
