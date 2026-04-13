package store

import (
	"path/filepath"
	"testing"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := NewFileStore(path)
	state := model.NewState()
	state.Profiles["p1"] = model.Profile{ID: "p1", Provider: "openai", Frontend: "codex", AuthMethod: "chatgpt", Protocol: "app_server", Enabled: true}
	if err := s.Save(state); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if _, ok := got.Profiles["p1"]; !ok {
		t.Fatalf("expected profile p1")
	}
}
