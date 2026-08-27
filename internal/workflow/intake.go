package workflow

import (
	"context"
	"strings"

	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/store"
)

func (s *Service) CreateBatch(ctx context.Context, input domain.CreateBatchInput) (BatchView, error) {
	input.Actor = strings.TrimSpace(input.Actor)
	now := s.now()
	batch := domain.ReleaseBatch{ID: s.newID("bat"), BatchCode: strings.TrimSpace(input.BatchCode), SiteCode: strings.TrimSpace(input.SiteCode), PlannedReleaseDate: input.PlannedReleaseDate, Owner: strings.TrimSpace(input.Owner), Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	for i := range input.Lots {
		input.Lots[i].ID = s.newID("lot")
		input.Lots[i].BatchID = batch.ID
		input.Lots[i].SpeciesCode = strings.TrimSpace(input.Lots[i].SpeciesCode)
	}
	if input.Actor == "" {
		return BatchView{}, domain.FieldError("actor", "操作人不能为空")
	}
	if err := domain.ValidateNewBatch(batch, input.Lots); err != nil {
		return BatchView{}, err
	}
	event := domain.NewAudit(batch.ID, input.Actor, "batch.created", map[string]any{"batchCode": batch.BatchCode, "speciesCount": len(input.Lots)}, now)
	snap, _, err := s.store.Create(ctx, batch, input.Lots, input.IdempotencyKey, store.Fingerprint(input), event)
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}

func (s *Service) SubmitEvidence(ctx context.Context, batchID string, input domain.EvidenceInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	input.SpeciesCode = strings.TrimSpace(input.SpeciesCode)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "evidence.submitted", map[string]any{"speciesCode": input.SpeciesCode}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureMutable(snap.Batch); err != nil {
			return err
		}
		if err := domain.EnsureStatus(snap.Batch, domain.StatusDraft); err != nil {
			return err
		}
		found := false
		candidate := make([]domain.SeedlingLot, len(snap.Lots))
		copy(candidate, snap.Lots)
		for i := range candidate {
			if candidate[i].SpeciesCode != input.SpeciesCode {
				continue
			}
			found = true
			candidate[i].NurseryOrigin = input.NurseryOrigin
			candidate[i].PermitDigest = input.PermitDigest
			candidate[i].QuarantineResult = input.QuarantineResult
			candidate[i].HandoverAt = input.HandoverAt
			candidate[i].EvidenceRefs = append([]string(nil), input.EvidenceRefs...)
		}
		if !found {
			return domain.NewError(domain.CodeNotFound, "物种配额不存在")
		}
		for i := range candidate {
			if candidate[i].SpeciesCode == input.SpeciesCode {
				normalized, err := domain.NormalizeEvidence(candidate[i], snap.Batch.PlannedReleaseDate, now, candidate)
				if err != nil {
					return err
				}
				candidate[i] = normalized
				event.Details["handoverAt"] = normalized.HandoverAt
				event.Details["evidenceRefs"] = normalized.EvidenceRefs
			}
		}
		snap.Lots = candidate
		complete := true
		for _, lot := range snap.Lots {
			if domain.ValidateEvidence(lot) != nil {
				complete = false
				break
			}
		}
		if complete {
			return domain.Transition(&snap.Batch, domain.StatusEvidenceReady)
		}
		return nil
	})
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}
