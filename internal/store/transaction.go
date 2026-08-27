package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"wetland-release-workbench/internal/domain"
)

type Mutator func(*domain.BatchSnapshot) error

func Fingerprint(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validateKey(key string) error {
	key = strings.TrimSpace(key)
	if len(key) < 8 || len(key) > 100 {
		return domain.FieldError("idempotencyKey", "长度必须为 8 到 100")
	}
	return nil
}

func (s *Store) Create(ctx context.Context, batch domain.ReleaseBatch, lots []domain.SeedlingLot, key, fingerprint string, event domain.AuditEvent) (domain.BatchSnapshot, bool, error) {
	if err := validateKey(key); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	defer tx.Rollback()
	if snap, replay, err := loadReplay(ctx, tx, key, fingerprint); replay || err != nil {
		return snap, replay, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO batches(id,batch_code,site_code,planned_release_date,owner,status,version,created_at,updated_at,submitted_by,review_note) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, batch.ID, batch.BatchCode, batch.SiteCode, batch.PlannedReleaseDate, batch.Owner, batch.Status, batch.Version, batch.CreatedAt.Format(time.RFC3339Nano), batch.UpdatedAt.Format(time.RFC3339Nano), batch.SubmittedBy, batch.ReviewNote)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return domain.BatchSnapshot{}, false, domain.NewError(domain.CodeConflict, "批次编号已存在")
		}
		return domain.BatchSnapshot{}, false, err
	}
	snap := domain.BatchSnapshot{Batch: batch, Lots: lots, Audit: []domain.AuditEvent{event}}
	if err = replaceLots(ctx, tx, snap); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = insertAudit(ctx, tx, event); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = saveIdempotency(ctx, tx, key, fingerprint, snap); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	return snap, false, nil
}

func (s *Store) Update(ctx context.Context, id string, expected int64, key, fingerprint string, event domain.AuditEvent, mutate Mutator) (domain.BatchSnapshot, bool, error) {
	if err := validateKey(key); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	defer tx.Rollback()
	if snap, replay, err := loadReplay(ctx, tx, key, fingerprint); replay || err != nil {
		return snap, replay, err
	}
	snap, err := loadSnapshot(ctx, tx, id)
	if err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if snap.Batch.Version != expected {
		return domain.BatchSnapshot{}, false, domain.NewError(domain.CodeVersionConflict, fmt.Sprintf("版本已变化，当前版本为 %d", snap.Batch.Version))
	}
	if err = mutate(&snap); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	snap.Batch.Version++
	snap.Batch.UpdatedAt = event.CreatedAt
	snap.Audit = append(snap.Audit, event)
	if err = saveSnapshot(ctx, tx, snap); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = insertAudit(ctx, tx, event); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = saveIdempotency(ctx, tx, key, fingerprint, snap); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	return snap, false, nil
}

func loadReplay(ctx context.Context, tx *sql.Tx, key, fingerprint string) (domain.BatchSnapshot, bool, error) {
	var old, result string
	err := tx.QueryRowContext(ctx, `SELECT fingerprint,result_json FROM idempotency WHERE key=?`, key).Scan(&old, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.BatchSnapshot{}, false, nil
	}
	if err != nil {
		return domain.BatchSnapshot{}, false, err
	}
	if old != fingerprint {
		return domain.BatchSnapshot{}, false, domain.NewError(domain.CodeIdempotency, "幂等键已用于不同请求")
	}
	var snap domain.BatchSnapshot
	if err = json.Unmarshal([]byte(result), &snap); err != nil {
		return snap, false, err
	}
	return snap, true, nil
}

func saveIdempotency(ctx context.Context, tx *sql.Tx, key, fingerprint string, snap domain.BatchSnapshot) error {
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO idempotency(key,fingerprint,batch_id,result_json,created_at) VALUES(?,?,?,?,?)`, key, fingerprint, snap.Batch.ID, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func insertAudit(ctx context.Context, tx *sql.Tx, e domain.AuditEvent) error {
	raw, _ := json.Marshal(e.Details)
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(batch_id,actor,action,details,created_at) VALUES(?,?,?,?,?)`, e.BatchID, e.Actor, e.Action, string(raw), e.CreatedAt.Format(time.RFC3339Nano))
	return err
}
