package router

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/policy"
)

type Input struct {
	Profiles []model.Profile
	Health   map[string]model.HealthSnapshot
	Policies []model.PolicyRule
	Now      time.Time
	Request  model.TaskRequest
}

func PickBest(in Input) model.RouteDecision {
	candidates, rejected := Rank(in)
	if len(candidates) == 0 {
		return model.RouteDecision{Score: 0, Rejected: rejected}
	}
	best := candidates[0]
	best.Rejected = rejected
	return best
}

func Rank(in Input) ([]model.RouteDecision, []string) {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	candidates := make([]model.RouteDecision, 0, len(in.Profiles))
	rejected := make([]string, 0, len(in.Profiles))
	for _, p := range in.Profiles {
		if !p.Enabled {
			rejected = append(rejected, fmt.Sprintf("%s: disabled", p.ID))
			continue
		}
		if in.Request.Frontend != "" && p.Frontend != in.Request.Frontend {
			rejected = append(rejected, fmt.Sprintf("%s: frontend mismatch", p.ID))
			continue
		}
		if !p.CooldownUntil.IsZero() && now.Before(p.CooldownUntil) {
			rejected = append(rejected, fmt.Sprintf("%s: cooldown until %s", p.ID, p.CooldownUntil.Format(time.RFC3339)))
			continue
		}
		if in.Request.RequiredProtocol != "" && p.Protocol != in.Request.RequiredProtocol {
			rejected = append(rejected, fmt.Sprintf("%s: protocol mismatch", p.ID))
			continue
		}
		if len(in.Request.PreferredProviders) > 0 && !slices.Contains(in.Request.PreferredProviders, p.Provider) {
			rejected = append(rejected, fmt.Sprintf("%s: provider not preferred", p.ID))
			continue
		}
		if len(in.Request.RequireTags) > 0 && !hasAllTags(p.Tags, in.Request.RequireTags) {
			rejected = append(rejected, fmt.Sprintf("%s: missing required tags", p.ID))
			continue
		}

		pol := policy.Evaluate(in.Policies, p, in.Request)
		if !pol.Allowed {
			rejected = append(rejected, fmt.Sprintf("%s: %s", p.ID, joinReasons(pol.Reasons)))
			continue
		}

		h := in.Health[p.ID]
		score := computeScore(p, h)
		candidates = append(candidates, model.RouteDecision{
			ProfileID: p.ID,
			Score:     score,
			Reasons: []string{
				fmt.Sprintf("provider=%s", p.Provider),
				fmt.Sprintf("protocol=%s", p.Protocol),
				fmt.Sprintf("priority=%d", p.Priority),
				fmt.Sprintf("remaining5m=%d", h.RemainingRequests5Min),
				fmt.Sprintf("latency=%dms", h.EstimatedLatencyMS),
				fmt.Sprintf("error_rate=%.2f%%", h.RecentErrorRatePercent),
			},
		})
	}
	slices.SortStableFunc(candidates, func(a, b model.RouteDecision) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		if a.ProfileID < b.ProfileID {
			return -1
		}
		if a.ProfileID > b.ProfileID {
			return 1
		}
		return 0
	})
	return candidates, rejected
}

func computeScore(p model.Profile, h model.HealthSnapshot) float64 {
	priorityBoost := float64(p.Priority) * 8.0
	quotaBoost := float64(h.RemainingRequests5Min)*1.5 + float64(h.RemainingRequestsHour)*0.15
	latencyPenalty := math.Min(float64(h.EstimatedLatencyMS)/25.0, 40)
	errorPenalty := math.Min(h.RecentErrorRatePercent*2.0, 60)
	budgetPenalty := p.BudgetDailyUSD * 0.2
	return 100 + priorityBoost + quotaBoost - latencyPenalty - errorPenalty - budgetPenalty
}

func hasAllTags(profileTags, required []string) bool {
	for _, req := range required {
		if !slices.Contains(profileTags, req) {
			return false
		}
	}
	return true
}

func joinReasons(rs []string) string {
	if len(rs) == 0 {
		return "policy denied"
	}
	return rs[0]
}
