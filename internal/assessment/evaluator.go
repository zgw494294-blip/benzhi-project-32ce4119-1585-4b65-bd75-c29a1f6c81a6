package assessment

import (
	"fmt"
	"sort"
	"time"

	"wetland-release-workbench/internal/domain"
)

func Evaluate(plans []domain.AcclimationPlan, observations []domain.Observation) Result {
	result := Result{Ready: len(plans) > 0}
	for _, plan := range latestPlans(plans) {
		matching := observationsFor(plan, observations)
		metric, findings := evaluateSpecies(plan, matching)
		result.Metrics = append(result.Metrics, metric)
		result.Findings = append(result.Findings, findings...)
		if !metric.Passed {
			result.Ready = false
		}
	}
	if len(result.Findings) > 0 {
		result.Ready = false
	}
	return result
}

func latestPlans(plans []domain.AcclimationPlan) []domain.AcclimationPlan {
	bySpecies := map[string]domain.AcclimationPlan{}
	lockedSpecies := map[string]bool{}
	for _, p := range plans {
		if p.LockedAt != nil {
			lockedSpecies[p.SpeciesCode] = true
		}
	}
	for _, p := range plans {
		if lockedSpecies[p.SpeciesCode] && p.LockedAt == nil {
			continue
		}
		if old, ok := bySpecies[p.SpeciesCode]; !ok || p.Revision > old.Revision {
			bySpecies[p.SpeciesCode] = p
		}
	}
	keys := make([]string, 0, len(bySpecies))
	for key := range bySpecies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]domain.AcclimationPlan, 0, len(keys))
	for _, key := range keys {
		out = append(out, bySpecies[key])
	}
	return out
}

func observationsFor(plan domain.AcclimationPlan, all []domain.Observation) []domain.Observation {
	var out []domain.Observation
	for _, o := range all {
		if o.SpeciesCode == plan.SpeciesCode && o.PlanRevision == plan.Revision && o.ZoneCode == plan.ZoneCode {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ObservedOn < out[j].ObservedOn })
	return out
}

func evaluateSpecies(plan domain.AcclimationPlan, observations []domain.Observation) (SpeciesMetrics, []Finding) {
	m := SpeciesMetrics{SpeciesCode: plan.SpeciesCode, ObservationDays: len(observations)}
	var findings []Finding
	if len(observations) == 0 {
		findings = append(findings, Finding{plan.SpeciesCode, "WINDOW_MISSING", "high", "尚未提交驯化观察", 1})
		return m, findings
	}
	window := latestContinuousWindow(observations)
	m.WindowDays = len(window)
	for _, o := range window {
		m.Sampled += o.SampledCount
		m.Surviving += o.SurvivingCount
		m.Diseased += o.DiseasedCount
		if o.Salinity < plan.SalinityRange.Min || o.Salinity > plan.SalinityRange.Max {
			findings = appendUnique(findings, Finding{plan.SpeciesCode, "SALINITY_OUT_OF_RANGE", "medium", fmt.Sprintf("盐度 %.2f 超出 %.2f-%.2f", o.Salinity, plan.SalinityRange.Min, plan.SalinityRange.Max), 3})
		}
		if o.WaterLevel < plan.WaterLevelRange.Min || o.WaterLevel > plan.WaterLevelRange.Max {
			findings = appendUnique(findings, Finding{plan.SpeciesCode, "WATER_LEVEL_OUT_OF_RANGE", "medium", fmt.Sprintf("水位 %.2f 超出 %.2f-%.2f", o.WaterLevel, plan.WaterLevelRange.Min, plan.WaterLevelRange.Max), 3})
		}
	}
	if m.Sampled > 0 {
		m.SurvivalRate = float64(m.Surviving) / float64(m.Sampled)
		m.DiseaseRate = float64(m.Diseased) / float64(m.Sampled)
	}
	if m.WindowDays < plan.DurationDays {
		findings = append(findings, Finding{plan.SpeciesCode, "WINDOW_INCOMPLETE", "high", fmt.Sprintf("连续观察窗口 %d 天，要求 %d 天", m.WindowDays, plan.DurationDays), 1})
	}
	if m.SurvivalRate < plan.MinimumSurvivalRate {
		findings = append(findings, Finding{plan.SpeciesCode, "SURVIVAL_BELOW_MINIMUM", "critical", fmt.Sprintf("成活率 %.2f%% 低于 %.2f%%", m.SurvivalRate*100, plan.MinimumSurvivalRate*100), 2})
	}
	if m.DiseaseRate > plan.MaximumDiseaseRate {
		findings = append(findings, Finding{plan.SpeciesCode, "DISEASE_ABOVE_MAXIMUM", "critical", fmt.Sprintf("病害率 %.2f%% 高于 %.2f%%", m.DiseaseRate*100, plan.MaximumDiseaseRate*100), 2})
	}
	m.Passed = len(findings) == 0
	return m, findings
}

func latestContinuousWindow(observations []domain.Observation) []domain.Observation {
	if len(observations) < 2 {
		return observations
	}
	start := len(observations) - 1
	for start > 0 {
		current, currentErr := time.Parse("2006-01-02", observations[start].ObservedOn)
		previous, previousErr := time.Parse("2006-01-02", observations[start-1].ObservedOn)
		if currentErr != nil || previousErr != nil || current.Sub(previous) != 24*time.Hour {
			break
		}
		start--
	}
	return observations[start:]
}

func appendUnique(findings []Finding, f Finding) []Finding {
	for _, old := range findings {
		if old.RuleCode == f.RuleCode {
			return findings
		}
	}
	return append(findings, f)
}

func ObservationPassesRule(plan domain.AcclimationPlan, obs domain.Observation, rule string) bool {
	if obs.SampledCount <= 0 {
		return false
	}
	survival := float64(obs.SurvivingCount) / float64(obs.SampledCount)
	disease := float64(obs.DiseasedCount) / float64(obs.SampledCount)
	switch rule {
	case "SALINITY_OUT_OF_RANGE":
		return obs.Salinity >= plan.SalinityRange.Min && obs.Salinity <= plan.SalinityRange.Max
	case "WATER_LEVEL_OUT_OF_RANGE":
		return obs.WaterLevel >= plan.WaterLevelRange.Min && obs.WaterLevel <= plan.WaterLevelRange.Max
	case "SURVIVAL_BELOW_MINIMUM":
		return survival >= plan.MinimumSurvivalRate
	case "DISEASE_ABOVE_MAXIMUM":
		return disease <= plan.MaximumDiseaseRate
	case "WINDOW_INCOMPLETE", "WINDOW_MISSING":
		return false
	default:
		return false
	}
}
