package assessment

import (
	"sort"
	"time"

	"wetland-release-workbench/internal/domain"
)

type TrendPoint struct {
	ObservedOn       string  `json:"observedOn"`
	PlanRevision     int     `json:"planRevision"`
	SurvivalRate     float64 `json:"survivalRate"`
	DiseaseRate      float64 `json:"diseaseRate"`
	SalinityMargin   float64 `json:"salinityMargin"`
	WaterLevelMargin float64 `json:"waterLevelMargin"`
}

type Trend struct {
	SpeciesCode      string       `json:"speciesCode"`
	ZoneCode         string       `json:"zoneCode"`
	PlanRevision     int          `json:"planRevision"`
	DurationDays     int          `json:"durationDays"`
	Points           []TrendPoint `json:"points"`
	HistoricalPoints []TrendPoint `json:"historicalPoints"`
	MissingDates     []string     `json:"missingDates"`
	WindowResetDates []string     `json:"windowResetDates"`
	RemainingDays    int          `json:"remainingDays"`
	NextRecordDate   string       `json:"nextRecordDate"`
}

func BuildTrend(plan domain.AcclimationPlan, observations []domain.Observation, now time.Time) (Trend, error) {
	trend := Trend{SpeciesCode: plan.SpeciesCode, ZoneCode: plan.ZoneCode, PlanRevision: plan.Revision, DurationDays: plan.DurationDays, RemainingDays: plan.DurationDays}
	var matching []domain.Observation
	for _, obs := range observations {
		if obs.SpeciesCode == plan.SpeciesCode && obs.ZoneCode == plan.ZoneCode {
			if obs.PlanRevision == plan.Revision {
				matching = append(matching, obs)
			} else {
				if _, err := time.Parse("2006-01-02", obs.ObservedOn); err != nil {
					return Trend{}, domain.FieldError("observedOn", "持久化日期无效")
				}
				survival, disease := 0.0, 0.0
				if obs.SampledCount > 0 {
					survival = float64(obs.SurvivingCount) / float64(obs.SampledCount)
					disease = float64(obs.DiseasedCount) / float64(obs.SampledCount)
				}
				trend.HistoricalPoints = append(trend.HistoricalPoints, TrendPoint{ObservedOn: obs.ObservedOn, PlanRevision: obs.PlanRevision, SurvivalRate: survival, DiseaseRate: disease})
			}
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].ObservedOn < matching[j].ObservedOn })
	if len(matching) == 0 {
		trend.NextRecordDate = now.UTC().Format("2006-01-02")
		return trend, nil
	}
	for _, obs := range matching {
		if _, err := time.Parse("2006-01-02", obs.ObservedOn); err != nil {
			return Trend{}, domain.FieldError("observedOn", "持久化日期无效")
		}
		marginSal := obs.Salinity - plan.SalinityRange.Min
		if upper := plan.SalinityRange.Max - obs.Salinity; upper < marginSal {
			marginSal = upper
		}
		marginWater := obs.WaterLevel - plan.WaterLevelRange.Min
		if upper := plan.WaterLevelRange.Max - obs.WaterLevel; upper < marginWater {
			marginWater = upper
		}
		survival, disease := 0.0, 0.0
		if obs.SampledCount > 0 {
			survival = float64(obs.SurvivingCount) / float64(obs.SampledCount)
			disease = float64(obs.DiseasedCount) / float64(obs.SampledCount)
		}
		trend.Points = append(trend.Points, TrendPoint{ObservedOn: obs.ObservedOn, PlanRevision: obs.PlanRevision, SurvivalRate: survival, DiseaseRate: disease, SalinityMargin: marginSal, WaterLevelMargin: marginWater})
	}
	windowStart := 0
	for i := 1; i < len(matching); i++ {
		prev, _ := time.Parse("2006-01-02", matching[i-1].ObservedOn)
		cur, _ := time.Parse("2006-01-02", matching[i].ObservedOn)
		if cur.Sub(prev) != 24*time.Hour {
			trend.WindowResetDates = append(trend.WindowResetDates, matching[i].ObservedOn)
			windowStart = i
		}
		for d := prev.AddDate(0, 0, 1); d.Before(cur); d = d.AddDate(0, 0, 1) {
			trend.MissingDates = append(trend.MissingDates, d.Format("2006-01-02"))
		}
	}
	windowDays := len(matching) - windowStart
	trend.RemainingDays = plan.DurationDays - windowDays
	if trend.RemainingDays < 0 {
		trend.RemainingDays = 0
	}
	last, _ := time.Parse("2006-01-02", matching[len(matching)-1].ObservedOn)
	trend.NextRecordDate = last.AddDate(0, 0, 1).Format("2006-01-02")
	return trend, nil
}
