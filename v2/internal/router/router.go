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
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	best := model.RouteDecision{Score: -1, Reasons: []string{}, Rejected: []string{}}
	for _, p := range in.Profiles {
		if !p.Enabled {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: disabled", p.ID))
			continue
		}
		if !p.CooldownUntil.IsZero() && now.Before(p.CooldownUntil) {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: cooldown until %s", p.ID, p.CooldownUntil.Format(time.RFC3339)))
			continue
		}
		if in.Request.RequiredProtocol != "" && p.Protocol != in.Request.RequiredProtocol {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: protocol mismatch", p.ID))
			continue
		}
		if len(in.Request.PreferredProviders) > 0 && !slices.Contains(in.Request.PreferredProviders, p.Provider) {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: provider not preferred", p.ID))
			continue
		}
		if len(in.Request.RequireTags) > 0 && !hasAllTags(p.Tags, in.Request.RequireTags) {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: missing required tags", p.ID))
			continue
		}

		pol := policy.Evaluate(in.Policies, p, in.Request)
		if !pol.Allowed {
			best.Rejected = append(best.Rejected, fmt.Sprintf("%s: %s", p.ID, joinReasons(pol.Reasons)))
			continue
		}

		h := in.Health[p.ID]
		score := computeScore(p, h)
		if score > best.Score {
			best.ProfileID = p.ID
			best.Score = score
			best.Reasons = []string{
				fmt.Sprintf("provider=%s", p.Provider),
				fmt.Sprintf("protocol=%s", p.Protocol),
				fmt.Sprintf("priority=%d", p.Priority),
				fmt.Sprintf("remaining5m=%d", h.RemainingRequests5Min),
				fmt.Sprintf("latency=%dms", h.EstimatedLatencyMS),
				fmt.Sprintf("error_rate=%.2f%%", h.RecentErrorRatePercent),
			}
		}
	}
	if best.ProfileID == "" {
		best.Score = 0
	}
	return best
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
