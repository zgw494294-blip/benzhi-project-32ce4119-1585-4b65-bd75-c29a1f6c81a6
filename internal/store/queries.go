package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"wetland-release-workbench/internal/domain"
)

func (s *Store) RemediationItems(ctx context.Context) ([]domain.EvidenceRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT i.batch_id,i.id,b.batch_code,b.owner,i.species_code,COALESCE(l.quantity,0),i.severity,i.due_date,i.status FROM issues i JOIN batches b ON b.id=i.batch_id LEFT JOIN lots l ON l.batch_id=i.batch_id AND l.species_code=i.species_code WHERE b.status <> ? AND i.status IN (?,?)`, domain.StatusApproved, domain.IssueOpen, domain.IssueRemediated)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EvidenceRecord
	for rows.Next() {
		var item domain.EvidenceRecord
		if err := rows.Scan(&item.BatchID, &item.IssueID, &item.BatchCode, &item.Owner, &item.SpeciesCode, &item.SpeciesQuantity, &item.Severity, &item.DueDate, &item.Status); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListRemediationQueue(ctx context.Context) ([]domain.RemediationQueueItem, error) {
	return s.RemediationItems(ctx)
}

func (s *Store) Get(ctx context.Context, id string) (domain.BatchSnapshot, error) {
	return loadSnapshot(ctx, s.db, id)
}

func (s *Store) List(ctx context.Context) ([]domain.ReleaseBatch, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,batch_code,site_code,planned_release_date,owner,status,version,created_at,updated_at,submitted_by,review_note FROM batches ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.ReleaseBatch
	for rows.Next() {
		var b domain.ReleaseBatch
		var created, updated string
		if err := rows.Scan(&b.ID, &b.BatchCode, &b.SiteCode, &b.PlannedReleaseDate, &b.Owner, &b.Status, &b.Version, &created, &updated, &b.SubmittedBy, &b.ReviewNote); err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		b.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *Store) Credential(ctx context.Context, id string) (domain.ReleaseCredential, error) {
	var batchID string
	err := s.db.QueryRowContext(ctx, `SELECT batch_id FROM credentials WHERE id=?`, id).Scan(&batchID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReleaseCredential{}, domain.NewError(domain.CodeNotFound, "凭据不存在")
	}
	if err != nil {
		return domain.ReleaseCredential{}, fmt.Errorf("读取凭据: %w", err)
	}
	snap, err := s.Get(ctx, batchID)
	if err != nil {
		return domain.ReleaseCredential{}, err
	}
	if snap.Credential == nil {
		return domain.ReleaseCredential{}, domain.NewError(domain.CodeNotFound, "凭据不存在")
	}
	return *snap.Credential, nil
}
