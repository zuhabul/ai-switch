package policy

import (
	"testing"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

func TestEvaluateDenyProvider(t *testing.T) {
	rules := []model.PolicyRule{{
		Name:          "deny-xai",
		Priority:      10,
		DenyProviders: []string{"xai"},
	}}
	res := Evaluate(rules, model.Profile{Provider: "xai"}, model.TaskRequest{})
	if res.Allowed {
		t.Fatalf("expected deny")
	}
}

func TestEvaluateRequireTag(t *testing.T) {
	rules := []model.PolicyRule{{
		Name:          "prod-only",
		Priority:      10,
		RequireAnyTag: []string{"prod"},
	}}
	res := Evaluate(rules, model.Profile{Provider: "openai", Tags: []string{"staging"}}, model.TaskRequest{})
	if res.Allowed {
		t.Fatalf("expected deny")
	}
	res = Evaluate(rules, model.Profile{Provider: "openai", Tags: []string{"prod"}}, model.TaskRequest{})
	if !res.Allowed {
		t.Fatalf("expected allow")
	}
}
