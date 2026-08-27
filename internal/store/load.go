package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"wetland-release-workbench/internal/domain"
)

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadSnapshot(ctx context.Context, q queryer, id string) (domain.BatchSnapshot, error) {
	var out domain.BatchSnapshot
	var created, updated string
	err := q.QueryRowContext(ctx, `SELECT id,batch_code,site_code,planned_release_date,owner,status,version,created_at,updated_at,submitted_by,review_note FROM batches WHERE id=?`, id).Scan(
		&out.Batch.ID, &out.Batch.BatchCode, &out.Batch.SiteCode, &out.Batch.PlannedReleaseDate, &out.Batch.Owner,
		&out.Batch.Status, &out.Batch.Version, &created, &updated, &out.Batch.SubmittedBy, &out.Batch.ReviewNote)
	if errors.Is(err, sql.ErrNoRows) {
		return out, domain.NewError(domain.CodeNotFound, "批次不存在")
	}
	if err != nil {
		return out, fmt.Errorf("读取批次: %w", err)
	}
	out.Batch.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.Batch.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if err = loadLots(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadPlans(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadObservations(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadIssues(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadManifest(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadCredential(ctx, q, &out); err != nil {
		return out, err
	}
	if err = loadAudit(ctx, q, &out); err != nil {
		return out, err
	}
	return out, nil
}

func loadLots(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	rows, err := q.QueryContext(ctx, `SELECT id,batch_id,species_code,quantity,nursery_origin,permit_digest,quarantine_result,handover_at,evidence_refs FROM lots WHERE batch_id=? ORDER BY species_code`, out.Batch.ID)
	if err != nil {
		return fmt.Errorf("读取苗源: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lot domain.SeedlingLot
		var refs string
		if err := rows.Scan(&lot.ID, &lot.BatchID, &lot.SpeciesCode, &lot.Quantity, &lot.NurseryOrigin, &lot.PermitDigest, &lot.QuarantineResult, &lot.HandoverAt, &refs); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(refs), &lot.EvidenceRefs); err != nil {
			return err
		}
		out.Lots = append(out.Lots, lot)
	}
	return rows.Err()
}

func loadPlans(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	rows, err := q.QueryContext(ctx, `SELECT id,batch_id,species_code,revision,zone_code,duration_days,salinity_min,salinity_max,water_min,water_max,minimum_survival,maximum_disease,minimum_sample,locked_at FROM plans WHERE batch_id=? ORDER BY species_code,revision`, out.Batch.ID)
	if err != nil {
		return fmt.Errorf("读取方案: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.AcclimationPlan
		var locked sql.NullString
		if err := rows.Scan(&p.ID, &p.BatchID, &p.SpeciesCode, &p.Revision, &p.ZoneCode, &p.DurationDays, &p.SalinityRange.Min, &p.SalinityRange.Max, &p.WaterLevelRange.Min, &p.WaterLevelRange.Max, &p.MinimumSurvivalRate, &p.MaximumDiseaseRate, &p.MinimumSampleCount, &locked); err != nil {
			return err
		}
		if locked.Valid {
			parsed, _ := time.Parse(time.RFC3339Nano, locked.String)
			p.LockedAt = &parsed
		}
		out.Plans = append(out.Plans, p)
	}
	return rows.Err()
}

func loadObservations(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	rows, err := q.QueryContext(ctx, `SELECT id,batch_id,species_code,plan_revision,observed_on,zone_code,sampled_count,surviving_count,diseased_count,salinity,water_level,notes,submitted_by,created_at FROM observations WHERE batch_id=? ORDER BY observed_on,id`, out.Batch.ID)
	if err != nil {
		return fmt.Errorf("读取观察: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var o domain.Observation
		var created string
		if err := rows.Scan(&o.ID, &o.BatchID, &o.SpeciesCode, &o.PlanRevision, &o.ObservedOn, &o.ZoneCode, &o.SampledCount, &o.SurvivingCount, &o.DiseasedCount, &o.Salinity, &o.WaterLevel, &o.Notes, &o.SubmittedBy, &created); err != nil {
			return err
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out.Observations = append(out.Observations, o)
	}
	return rows.Err()
}

func loadIssues(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	rows, err := q.QueryContext(ctx, `SELECT id,batch_id,species_code,rule_code,severity,description,due_date,status,remediation,verification_observation_id,closed_at FROM issues WHERE batch_id=? ORDER BY status,severity,rule_code`, out.Batch.ID)
	if err != nil {
		return fmt.Errorf("读取阻断项: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var issue domain.BlockingIssue
		var closed sql.NullString
		if err := rows.Scan(&issue.ID, &issue.BatchID, &issue.SpeciesCode, &issue.RuleCode, &issue.Severity, &issue.Description, &issue.DueDate, &issue.Status, &issue.Remediation, &issue.VerificationObservationID, &closed); err != nil {
			return err
		}
		if closed.Valid {
			parsed, _ := time.Parse(time.RFC3339Nano, closed.String)
			issue.ClosedAt = &parsed
		}
		out.Issues = append(out.Issues, issue)
	}
	return rows.Err()
}

func loadManifest(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	var m domain.FrozenManifest
	var items, frozen string
	err := q.QueryRowContext(ctx, `SELECT batch_id,batch_version,batch_code,site_code,evidence_summary,items,frozen_at FROM manifests WHERE batch_id=?`, out.Batch.ID).Scan(&m.BatchID, &m.BatchVersion, &m.BatchCode, &m.SiteCode, &m.EvidenceSummary, &items, &frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(items), &m.Items); err != nil {
		return err
	}
	m.FrozenAt, _ = time.Parse(time.RFC3339Nano, frozen)
	out.Manifest = &m
	return nil
}

func loadCredential(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	var c domain.ReleaseCredential
	var items, approved string
	err := q.QueryRowContext(ctx, `SELECT id,batch_id,manifest_digest,site_code,species_quantities,approved_at,approved_by,signature,key_id FROM credentials WHERE batch_id=?`, out.Batch.ID).Scan(&c.ID, &c.BatchID, &c.ManifestDigest, &c.SiteCode, &items, &approved, &c.ApprovedBy, &c.Signature, &c.KeyID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(items), &c.SpeciesQuantities); err != nil {
		return err
	}
	c.ApprovedAt, _ = time.Parse(time.RFC3339Nano, approved)
	out.Credential = &c
	return nil
}

func loadAudit(ctx context.Context, q queryer, out *domain.BatchSnapshot) error {
	rows, err := q.QueryContext(ctx, `SELECT id,batch_id,actor,action,details,created_at FROM audit_events WHERE batch_id=? ORDER BY created_at,id`, out.Batch.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.AuditEvent
		var details, created string
		if err := rows.Scan(&e.ID, &e.BatchID, &e.Actor, &e.Action, &details, &created); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(details), &e.Details)
		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out.Audit = append(out.Audit, e)
	}
	return rows.Err()
}
