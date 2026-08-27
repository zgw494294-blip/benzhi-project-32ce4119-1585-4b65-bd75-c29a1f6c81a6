package assessment

import (
	"testing"
	"time"

	"wetland-release-workbench/internal/domain"
)

func standardPlan() domain.AcclimationPlan {
	now := time.Now().UTC()
	return domain.AcclimationPlan{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", Revision: 1, DurationDays: 2, SalinityRange: domain.NumberRange{Min: 5, Max: 15}, WaterLevelRange: domain.NumberRange{Min: 0, Max: 2}, MinimumSurvivalRate: .8, MaximumDiseaseRate: .1, MinimumSampleCount: 10, LockedAt: &now}
}

func TestEvaluateContinuousWindowPasses(t *testing.T) {
	observations := []domain.Observation{
		{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: "2026-08-25", SampledCount: 20, SurvivingCount: 19, DiseasedCount: 1, Salinity: 10, WaterLevel: 1},
		{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: "2026-08-26", SampledCount: 20, SurvivingCount: 18, DiseasedCount: 1, Salinity: 11, WaterLevel: 1.1},
	}
	result := Evaluate([]domain.AcclimationPlan{standardPlan()}, observations)
	if !result.Ready || len(result.Findings) != 0 {
		t.Fatalf("合格窗口应通过：%+v", result)
	}
}

func TestEvaluateMapsThresholdFailures(t *testing.T) {
	observations := []domain.Observation{{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: "2026-08-25", SampledCount: 20, SurvivingCount: 10, DiseasedCount: 5, Salinity: 22, WaterLevel: 3}}
	result := Evaluate([]domain.AcclimationPlan{standardPlan()}, observations)
	codes := map[string]bool{}
	for _, finding := range result.Findings {
		codes[finding.RuleCode] = true
	}
	for _, expected := range []string{"WINDOW_INCOMPLETE", "SURVIVAL_BELOW_MINIMUM", "DISEASE_ABOVE_MAXIMUM", "SALINITY_OUT_OF_RANGE", "WATER_LEVEL_OUT_OF_RANGE"} {
		if !codes[expected] {
			t.Errorf("缺少失败规则 %s", expected)
		}
	}
}

func TestEvaluateRejectsGapsInDailyWindow(t *testing.T) {
	observations := []domain.Observation{
		{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: "2026-08-20", SampledCount: 20, SurvivingCount: 19, Salinity: 10, WaterLevel: 1},
		{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: "2026-08-22", SampledCount: 20, SurvivingCount: 19, Salinity: 10, WaterLevel: 1},
	}
	result := Evaluate([]domain.AcclimationPlan{standardPlan()}, observations)
	if result.Ready || result.Metrics[0].WindowDays != 1 {
		t.Fatalf("日期有缺口时不应形成连续窗口：%+v", result)
	}
}

func TestCredentialDetectsTampering(t *testing.T) {
	manifest := domain.FrozenManifest{BatchID: "bat-1", BatchVersion: 8, BatchCode: "WET-001", SiteCode: "SITE-A", Items: []domain.ManifestItem{{SpeciesCode: "SPECIES-A", Quantity: 10}}, FrozenAt: time.Date(2026, 8, 27, 1, 2, 3, 0, time.UTC)}
	signer := NewSigner("key-1", []byte("test-secret-key"))
	credential := signer.Issue("cred-1", manifest, "复核人", manifest.FrozenAt)
	if !signer.Verify(credential) {
		t.Fatal("原始凭据应通过")
	}
	credential.SiteCode = "SITE-B"
	if signer.Verify(credential) {
		t.Fatal("篡改地块后不应通过")
	}
}

func TestEvidenceDigestIsOrderIndependent(t *testing.T) {
	first := domain.BatchSnapshot{Lots: []domain.SeedlingLot{{ID: "2", SpeciesCode: "B", Quantity: 2}, {ID: "1", SpeciesCode: "A", Quantity: 1}}}
	second := domain.BatchSnapshot{Lots: []domain.SeedlingLot{{ID: "1", SpeciesCode: "A", Quantity: 1}, {ID: "2", SpeciesCode: "B", Quantity: 2}}}
	if EvidenceDigest(first) != EvidenceDigest(second) {
		t.Fatal("同一证据集合的摘要不应受读取顺序影响")
	}
	second.Lots[0].Quantity = 99
	if EvidenceDigest(first) == EvidenceDigest(second) {
		t.Fatal("证据事实改变后摘要必须改变")
	}
}
