package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"wetland-release-workbench/internal/domain"
)

func saveSnapshot(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	b := snap.Batch
	result, err := tx.ExecContext(ctx, `UPDATE batches SET site_code=?,planned_release_date=?,owner=?,status=?,version=?,updated_at=?,submitted_by=?,review_note=? WHERE id=?`, b.SiteCode, b.PlannedReleaseDate, b.Owner, b.Status, b.Version, b.UpdatedAt.Format(time.RFC3339Nano), b.SubmittedBy, b.ReviewNote, b.ID)
	if err != nil {
		return fmt.Errorf("保存批次: %w", err)
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return domain.NewError(domain.CodeNotFound, "批次不存在")
	}
	if err := replaceLots(ctx, tx, snap); err != nil {
		return err
	}
	if err := replacePlans(ctx, tx, snap); err != nil {
		return err
	}
	if err := replaceObservations(ctx, tx, snap); err != nil {
		return err
	}
	if err := replaceIssues(ctx, tx, snap); err != nil {
		return err
	}
	if err := replaceManifest(ctx, tx, snap); err != nil {
		return err
	}
	if err := replaceCredential(ctx, tx, snap); err != nil {
		return err
	}
	return nil
}

func replaceLots(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM lots WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	for _, v := range snap.Lots {
		refs, _ := json.Marshal(v.EvidenceRefs)
		if _, err := tx.ExecContext(ctx, `INSERT INTO lots(id,batch_id,species_code,quantity,nursery_origin,permit_digest,quarantine_result,handover_at,evidence_refs) VALUES(?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.SpeciesCode, v.Quantity, v.NurseryOrigin, v.PermitDigest, v.QuarantineResult, v.HandoverAt, string(refs)); err != nil {
			return fmt.Errorf("保存苗源: %w", err)
		}
	}
	return nil
}

func replacePlans(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM plans WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	for _, v := range snap.Plans {
		var locked any
		if v.LockedAt != nil {
			locked = v.LockedAt.Format(time.RFC3339Nano)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO plans(id,batch_id,species_code,revision,zone_code,duration_days,salinity_min,salinity_max,water_min,water_max,minimum_survival,maximum_disease,minimum_sample,locked_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.SpeciesCode, v.Revision, v.ZoneCode, v.DurationDays, v.SalinityRange.Min, v.SalinityRange.Max, v.WaterLevelRange.Min, v.WaterLevelRange.Max, v.MinimumSurvivalRate, v.MaximumDiseaseRate, v.MinimumSampleCount, locked); err != nil {
			return fmt.Errorf("保存方案: %w", err)
		}
	}
	return nil
}

func replaceObservations(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM observations WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	for _, v := range snap.Observations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO observations(id,batch_id,species_code,plan_revision,observed_on,zone_code,sampled_count,surviving_count,diseased_count,salinity,water_level,notes,submitted_by,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.SpeciesCode, v.PlanRevision, v.ObservedOn, v.ZoneCode, v.SampledCount, v.SurvivingCount, v.DiseasedCount, v.Salinity, v.WaterLevel, v.Notes, v.SubmittedBy, v.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("保存观察: %w", err)
		}
	}
	return nil
}

func replaceIssues(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM issues WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	for _, v := range snap.Issues {
		var closed any
		if v.ClosedAt != nil {
			closed = v.ClosedAt.Format(time.RFC3339Nano)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO issues(id,batch_id,species_code,rule_code,severity,description,due_date,status,remediation,verification_observation_id,closed_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.SpeciesCode, v.RuleCode, v.Severity, v.Description, v.DueDate, v.Status, v.Remediation, v.VerificationObservationID, closed); err != nil {
			return fmt.Errorf("保存阻断项: %w", err)
		}
	}
	return nil
}

func replaceManifest(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM manifests WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	if snap.Manifest == nil {
		return nil
	}
	v := snap.Manifest
	items, _ := json.Marshal(v.Items)
	_, err := tx.ExecContext(ctx, `INSERT INTO manifests(batch_id,batch_version,batch_code,site_code,evidence_summary,items,frozen_at) VALUES(?,?,?,?,?,?,?)`, v.BatchID, v.BatchVersion, v.BatchCode, v.SiteCode, v.EvidenceSummary, string(items), v.FrozenAt.Format(time.RFC3339Nano))
	return err
}

func replaceCredential(ctx context.Context, tx *sql.Tx, snap domain.BatchSnapshot) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM credentials WHERE batch_id=?`, snap.Batch.ID); err != nil {
		return err
	}
	if snap.Credential == nil {
		return nil
	}
	v := snap.Credential
	items, _ := json.Marshal(v.SpeciesQuantities)
	_, err := tx.ExecContext(ctx, `INSERT INTO credentials(id,batch_id,manifest_digest,site_code,species_quantities,approved_at,approved_by,signature,key_id) VALUES(?,?,?,?,?,?,?,?,?)`, v.ID, v.BatchID, v.ManifestDigest, v.SiteCode, string(items), v.ApprovedAt.Format(time.RFC3339Nano), v.ApprovedBy, v.Signature, v.KeyID)
	return err
}
