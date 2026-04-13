package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/api"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/store"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4417", "listen address")
	statePath := flag.String("state", defaultStatePath(), "state file path")
	flag.Parse()

	st := store.NewFileStore(*statePath)
	svc := service.New(st)
	if err := svc.Init(context.Background()); err != nil {
		log.Fatalf("init failed: %v", err)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           api.NewServer(svc).Handler(),
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
