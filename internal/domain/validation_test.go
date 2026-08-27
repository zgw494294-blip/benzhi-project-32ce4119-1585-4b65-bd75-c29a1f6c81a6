package domain

import (
	"testing"
	"time"
)

func TestValidateNewBatchRejectsDuplicateSpecies(t *testing.T) {
	batch := ReleaseBatch{BatchCode: "WET-001", SiteCode: "SITE-A", PlannedReleaseDate: "2026-09-01", Owner: "交接员"}
	lots := []SeedlingLot{{SpeciesCode: "SPECIES-A", Quantity: 10}, {SpeciesCode: "SPECIES-A", Quantity: 20}}
	if code := ErrorCodeOf(ValidateNewBatch(batch, lots)); code != CodeConflict {
		t.Fatalf("期望重复配额冲突，得到 %s", code)
	}
}

func TestValidateObservationRules(t *testing.T) {
	plan := AcclimationPlan{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", Revision: 1, MinimumSampleCount: 10}
	observation := Observation{SpeciesCode: "SPECIES-A", ZoneCode: "ZONE-A", PlanRevision: 1, ObservedOn: time.Now().UTC().Format("2006-01-02"), SampledCount: 9, SurvivingCount: 9, SubmittedBy: "技术员"}
	if err := ValidateObservation(observation, plan, nil); err == nil {
		t.Fatal("低于最低抽样数应被拒绝")
	}
	observation.SampledCount = 10
	observation.SurvivingCount = 11
	if err := ValidateObservation(observation, plan, nil); err == nil {
		t.Fatal("存活数超过抽样数应被拒绝")
	}
}

func TestStateTransitionsAndFrozenGuard(t *testing.T) {
	batch := ReleaseBatch{Status: StatusDraft}
	if err := Transition(&batch, StatusEvidenceReady); err != nil {
		t.Fatalf("合法迁移失败：%v", err)
	}
	if err := Transition(&batch, StatusApproved); err == nil {
		t.Fatal("跳过驯化与复核不应批准")
	}
	batch.Status = StatusApproved
	if code := ErrorCodeOf(EnsureMutable(batch)); code != CodeFrozen {
		t.Fatalf("冻结保护错误：%s", code)
	}
}
