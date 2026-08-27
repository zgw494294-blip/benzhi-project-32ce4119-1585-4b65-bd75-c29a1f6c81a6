package store

import (
	"context"
	"fmt"
)

const schemaVersion = 1

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_meta (version INTEGER NOT NULL)`,
		`INSERT INTO schema_meta(version) SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM schema_meta)`,
		`CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY, batch_code TEXT NOT NULL UNIQUE, site_code TEXT NOT NULL,
			planned_release_date TEXT NOT NULL, owner TEXT NOT NULL, status TEXT NOT NULL,
			version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			submitted_by TEXT NOT NULL DEFAULT '', review_note TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE IF NOT EXISTS lots (
			id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, species_code TEXT NOT NULL, quantity INTEGER NOT NULL,
			nursery_origin TEXT NOT NULL, permit_digest TEXT NOT NULL, quarantine_result TEXT NOT NULL,
			handover_at TEXT NOT NULL, evidence_refs TEXT NOT NULL,
			UNIQUE(batch_id, species_code), FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, species_code TEXT NOT NULL, revision INTEGER NOT NULL,
			zone_code TEXT NOT NULL, duration_days INTEGER NOT NULL, salinity_min REAL NOT NULL, salinity_max REAL NOT NULL,
			water_min REAL NOT NULL, water_max REAL NOT NULL, minimum_survival REAL NOT NULL,
			maximum_disease REAL NOT NULL, minimum_sample INTEGER NOT NULL, locked_at TEXT,
			UNIQUE(batch_id, species_code, revision), FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS observations (
			id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, species_code TEXT NOT NULL, plan_revision INTEGER NOT NULL,
			observed_on TEXT NOT NULL, zone_code TEXT NOT NULL, sampled_count INTEGER NOT NULL,
			surviving_count INTEGER NOT NULL, diseased_count INTEGER NOT NULL, salinity REAL NOT NULL,
			water_level REAL NOT NULL, notes TEXT NOT NULL, submitted_by TEXT NOT NULL, created_at TEXT NOT NULL,
			UNIQUE(batch_id, species_code, zone_code, observed_on), FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS issues (
			id TEXT PRIMARY KEY, batch_id TEXT NOT NULL, species_code TEXT NOT NULL, rule_code TEXT NOT NULL,
			severity TEXT NOT NULL, description TEXT NOT NULL, due_date TEXT NOT NULL, status TEXT NOT NULL,
			remediation TEXT NOT NULL, verification_observation_id TEXT NOT NULL, closed_at TEXT,
			UNIQUE(batch_id, species_code, rule_code), FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS manifests (
			batch_id TEXT PRIMARY KEY, batch_version INTEGER NOT NULL, batch_code TEXT NOT NULL, site_code TEXT NOT NULL,
			evidence_summary TEXT NOT NULL, items TEXT NOT NULL, frozen_at TEXT NOT NULL,
			FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY, batch_id TEXT NOT NULL UNIQUE, manifest_digest TEXT NOT NULL, site_code TEXT NOT NULL,
			species_quantities TEXT NOT NULL, approved_at TEXT NOT NULL, approved_by TEXT NOT NULL,
			signature TEXT NOT NULL, key_id TEXT NOT NULL, FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT, batch_id TEXT NOT NULL, actor TEXT NOT NULL,
			action TEXT NOT NULL, details TEXT NOT NULL, created_at TEXT NOT NULL,
			FOREIGN KEY(batch_id) REFERENCES batches(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS audit_batch_time ON audit_events(batch_id, created_at, id)`,
		`CREATE TABLE IF NOT EXISTS idempotency (
			key TEXT PRIMARY KEY, fingerprint TEXT NOT NULL, batch_id TEXT NOT NULL,
			result_json TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`UPDATE schema_meta SET version = 1 WHERE version < 1`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行 schemaVersion %d 迁移: %w", schemaVersion, err)
		}
	}
	return nil
}
