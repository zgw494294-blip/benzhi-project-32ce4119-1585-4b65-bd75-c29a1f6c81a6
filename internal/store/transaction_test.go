package store

import (
	"context"
	"testing"
	"time"

	"wetland-release-workbench/internal/domain"
)

func TestCreateReplayAndVersionConflict(t *testing.T) {
	s, err := Open("file:store-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	batch := domain.ReleaseBatch{ID: "bat-1", BatchCode: "WET-001", SiteCode: "SITE-A", PlannedReleaseDate: "2026-09-01", Owner: "交接员", Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	lots := []domain.SeedlingLot{{ID: "lot-1", BatchID: batch.ID, SpeciesCode: "SPECIES-A", Quantity: 20}}
	event := domain.NewAudit(batch.ID, "交接员", "batch.created", nil, now)
	first, replay, err := s.Create(ctx, batch, lots, "create-key-0001", "fingerprint-a", event)
	if err != nil || replay {
		t.Fatalf("首次创建失败：replay=%v err=%v", replay, err)
	}
	second, replay, err := s.Create(ctx, batch, lots, "create-key-0001", "fingerprint-a", event)
	if err != nil || !replay || second.Batch.ID != first.Batch.ID {
		t.Fatalf("相同幂等请求应复用结果：replay=%v err=%v", replay, err)
	}
	_, _, err = s.Update(ctx, batch.ID, 99, "update-key-0001", "fingerprint-b", event, func(*domain.BatchSnapshot) error { return nil })
	if domain.ErrorCodeOf(err) != domain.CodeVersionConflict {
		t.Fatalf("期望版本冲突，得到 %v", err)
	}
}

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := t.TempDir() + "/recovery.db"
	now := time.Now().UTC()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	batch := domain.ReleaseBatch{ID: "bat-recover", BatchCode: "WET-REC", SiteCode: "SITE-R", PlannedReleaseDate: "2026-09-02", Owner: "负责人", Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	_, _, err = s.Create(context.Background(), batch, []domain.SeedlingLot{{ID: "lot-r", BatchID: batch.ID, SpeciesCode: "SPECIES-R", Quantity: 8}}, "recover-key-001", "recover-fp", domain.NewAudit(batch.ID, "负责人", "batch.created", nil, now))
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	summary, err := reopened.RecoverySummary(context.Background())
	if err != nil || summary.UnfinishedCount != 1 {
		t.Fatalf("恢复汇总不正确：%+v err=%v", summary, err)
	}
	loaded, err := reopened.Get(context.Background(), batch.ID)
	if err != nil || loaded.Batch.BatchCode != batch.BatchCode {
		t.Fatalf("重启后批次未恢复：%+v err=%v", loaded.Batch, err)
	}
}
