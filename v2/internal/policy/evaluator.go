package policy

import (
	"fmt"
	"slices"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

type Evaluation struct {
	Allowed bool
	Reasons []string
}

func Evaluate(rules []model.PolicyRule, profile model.Profile, req model.TaskRequest) Evaluation {
	eval := Evaluation{Allowed: true, Reasons: []string{}}
	if len(rules) == 0 {
		return eval
	}

	sorted := slices.Clone(rules)
	slices.SortStableFunc(sorted, func(a, b model.PolicyRule) int {
		if a.Priority > b.Priority {
			return -1
		}
		if a.Priority < b.Priority {
			return 1
		}
		return 0
	})

	for _, rule := range sorted {
		if !matchesFrontend(rule, req.Frontend) || !matchesTaskClass(rule, req.TaskClass) {
			continue
		}
		if len(rule.AllowProviders) > 0 && !slices.Contains(rule.AllowProviders, profile.Provider) {
			eval.Allowed = false
			eval.Reasons = append(eval.Reasons, fmt.Sprintf("policy %s disallowed provider %s", rule.Name, profile.Provider))
			continue
		}
		if slices.Contains(rule.DenyProviders, profile.Provider) {
			eval.Allowed = false
			eval.Reasons = append(eval.Reasons, fmt.Sprintf("policy %s denied provider %s", rule.Name, profile.Provider))
			continue
		}
		if len(rule.RequireAuthMethods) > 0 && !slices.Contains(rule.RequireAuthMethods, profile.AuthMethod) {
			eval.Allowed = false
			eval.Reasons = append(eval.Reasons, fmt.Sprintf("policy %s requires auth methods %v", rule.Name, rule.RequireAuthMethods))
			continue
		}
		if len(rule.RequireAnyTag) > 0 && !hasAnyTag(profile.Tags, rule.RequireAnyTag) {
			eval.Allowed = false
			eval.Reasons = append(eval.Reasons, fmt.Sprintf("policy %s requires one of tags %v", rule.Name, rule.RequireAnyTag))
			continue
		}
		if rule.MaxBudgetDailyUSD > 0 && profile.BudgetDailyUSD > rule.MaxBudgetDailyUSD {
			eval.Allowed = false
			eval.Reasons = append(eval.Reasons, fmt.Sprintf("policy %s budget %.2f > %.2f", rule.Name, profile.BudgetDailyUSD, rule.MaxBudgetDailyUSD))
			continue
		}
	}
	return eval
}

func matchesFrontend(rule model.PolicyRule, frontend string) bool {
	if len(rule.Frontends) == 0 {
		return true
	}
	return slices.Contains(rule.Frontends, frontend)
}

func matchesTaskClass(rule model.PolicyRule, taskClass string) bool {
	if len(rule.TaskClasses) == 0 {
		return true
	}
	return slices.Contains(rule.TaskClasses, taskClass)
}

func hasAnyTag(profileTags, required []string) bool {
	for _, t := range profileTags {
		if slices.Contains(required, t) {
			return true
		}
	}
	return false
}
