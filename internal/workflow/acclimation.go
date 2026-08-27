package workflow

import (
	"context"
	"time"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/store"
)

func (s *Service) SetPlan(ctx context.Context, batchID string, input domain.PlanInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "plan.configured", map[string]any{"speciesCode": input.Plan.SpeciesCode}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureMutable(snap.Batch); err != nil {
			return err
		}
		if err := domain.EnsureStatus(snap.Batch, domain.StatusEvidenceReady); err != nil {
			return err
		}
		known := false
		for _, lot := range snap.Lots {
			if lot.SpeciesCode == input.Plan.SpeciesCode {
				known = true
			}
		}
		if !known {
			return domain.NewError(domain.CodeNotFound, "物种配额不存在")
		}
		plan := input.Plan
		plan.ID = s.newID("pln")
		plan.BatchID = batchID
		plan.Revision = 1
		for _, old := range snap.Plans {
			if old.SpeciesCode == plan.SpeciesCode {
				if old.LockedAt != nil {
					return domain.NewError(domain.CodeState, "驯化开始后方案不可修订")
				}
				if old.Revision >= plan.Revision {
					plan.Revision = old.Revision + 1
				}
			}
		}
		if err := domain.ValidatePlan(plan); err != nil {
			return err
		}
		snap.Plans = append(snap.Plans, plan)
		covered := map[string]bool{}
		for _, p := range snap.Plans {
			covered[p.SpeciesCode] = true
		}
		if len(covered) == len(snap.Lots) {
			for i := range snap.Plans {
				locked := now
				snap.Plans[i].LockedAt = &locked
			}
			return domain.Transition(&snap.Batch, domain.StatusAcclimating)
		}
		return nil
	})
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}

func (s *Service) SubmitObservation(ctx context.Context, batchID string, input domain.ObservationInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "observation.submitted", map[string]any{"speciesCode": input.Observation.SpeciesCode, "observedOn": input.Observation.ObservedOn}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureMutable(snap.Batch); err != nil {
			return err
		}
		if err := domain.EnsureStatus(snap.Batch, domain.StatusAcclimating, domain.StatusNeedsRemediation); err != nil {
			return err
		}
		plan, ok := findPlan(snap.Plans, input.Observation.SpeciesCode, input.Observation.PlanRevision)
		if !ok {
			return domain.NewError(domain.CodeNotFound, "锁定方案不存在")
		}
		obs := input.Observation
		obs.ID = s.newID("obs")
		obs.BatchID = batchID
		obs.SubmittedBy = input.Meta.Actor
		obs.CreatedAt = now
		if err := domain.ValidateObservation(obs, plan, snap.Observations); err != nil {
			return err
		}
		snap.Observations = append(snap.Observations, obs)
		snap.Batch.SubmittedBy = input.Meta.Actor
		result := assessment.Evaluate(snap.Plans, snap.Observations)
		syncFindings(snap, result.Findings, s.newID, now)
		if len(result.Findings) > 0 {
			if snap.Batch.Status == domain.StatusAcclimating {
				return domain.Transition(&snap.Batch, domain.StatusNeedsRemediation)
			}
			return nil
		}
		if result.Ready && allIssuesClosed(snap.Issues) {
			return domain.Transition(&snap.Batch, domain.StatusReviewReady)
		}
		return nil
	})
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}

func findPlan(plans []domain.AcclimationPlan, species string, revision int) (domain.AcclimationPlan, bool) {
	for _, p := range plans {
		if p.SpeciesCode == species && p.Revision == revision {
			return p, true
		}
	}
	return domain.AcclimationPlan{}, false
}

func syncFindings(snap *domain.BatchSnapshot, findings []assessment.Finding, newID func(string) string, now time.Time) {
	for _, f := range findings {
		found := false
		for i := range snap.Issues {
			issue := &snap.Issues[i]
			if issue.SpeciesCode == f.SpeciesCode && issue.RuleCode == f.RuleCode {
				found = true
				if issue.Status == domain.IssueClosed {
					issue.Status = domain.IssueOpen
					issue.ClosedAt = nil
				}
				issue.Description = f.Description
				break
			}
		}
		if !found {
			due := now.AddDate(0, 0, f.DueDays).Format("2006-01-02")
			snap.Issues = append(snap.Issues, assessment.FindingToIssue(f, snap.Batch.ID, newID("iss"), due))
		}
	}
}

func allIssuesClosed(issues []domain.BlockingIssue) bool {
	for _, issue := range issues {
		if issue.Status != domain.IssueClosed {
			return false
		}
	}
	return true
}
