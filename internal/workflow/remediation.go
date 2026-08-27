package workflow

import (
	"context"
	"strings"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/store"
)

func (s *Service) Remediate(ctx context.Context, batchID, issueID string, input domain.RemediationInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	if strings.TrimSpace(input.Remediation) == "" {
		return BatchView{}, domain.FieldError("remediation", "整改措施不能为空")
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "issue.remediated", map[string]any{"issueID": issueID, "observationID": input.ObservationID}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureStatus(snap.Batch, domain.StatusNeedsRemediation); err != nil {
			return err
		}
		idx := -1
		for i := range snap.Issues {
			if snap.Issues[i].ID == issueID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return domain.NewError(domain.CodeNotFound, "阻断项不存在")
		}
		issue := &snap.Issues[idx]
		issue.Remediation = strings.TrimSpace(input.Remediation)
		issue.Status = domain.IssueRemediated
		if input.ObservationID != "" {
			var obs domain.Observation
			found := false
			for _, candidate := range snap.Observations {
				if candidate.ID == input.ObservationID {
					obs = candidate
					found = true
					break
				}
			}
			if !found {
				return domain.NewError(domain.CodeNotFound, "复验观察不存在")
			}
			if obs.SpeciesCode != issue.SpeciesCode {
				return domain.NewError(domain.CodeInvalid, "复验观察与阻断物种不一致")
			}
			plan, ok := findPlan(snap.Plans, obs.SpeciesCode, obs.PlanRevision)
			if !ok {
				return domain.NewError(domain.CodeNotFound, "复验方案不存在")
			}
			passes := assessment.ObservationPassesRule(plan, obs, issue.RuleCode)
			if issue.RuleCode == "WINDOW_INCOMPLETE" || issue.RuleCode == "WINDOW_MISSING" {
				passes = !findingPresent(assessment.Evaluate(snap.Plans, snap.Observations), issue.SpeciesCode, issue.RuleCode)
			}
			if !passes {
				return domain.NewError(domain.CodeInvalid, "复验观察尚未满足关闭条件")
			}
			issue.VerificationObservationID = obs.ID
			issue.Status = domain.IssueClosed
			closed := now
			issue.ClosedAt = &closed
		}
		if assessment.Evaluate(snap.Plans, snap.Observations).Ready && allIssuesClosed(snap.Issues) {
			return domain.Transition(&snap.Batch, domain.StatusReviewReady)
		}
		return nil
	})
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}

func (s *Service) JointRemediate(ctx context.Context, batchID string, input domain.JointRemediationInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	if strings.TrimSpace(input.Remediation) == "" {
		return BatchView{}, domain.FieldError("remediation", "整改措施不能为空")
	}
	if len(input.IssueIDs) == 0 {
		return BatchView{}, domain.FieldError("issueIDs", "至少选择一个阻断项")
	}
	if strings.TrimSpace(input.ObservationID) == "" {
		return BatchView{}, domain.FieldError("observationID", "复验观察不能为空")
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "issues.joint_remediated", map[string]any{"issueIDs": input.IssueIDs, "observationID": input.ObservationID}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureStatus(snap.Batch, domain.StatusNeedsRemediation); err != nil {
			return err
		}
		seen := map[string]bool{}
		var selected []*domain.BlockingIssue
		species := ""
		for _, id := range input.IssueIDs {
			if seen[id] {
				return domain.NewError(domain.CodeConflict, "阻断项不可重复选择")
			}
			seen[id] = true
			var found *domain.BlockingIssue
			for i := range snap.Issues {
				if snap.Issues[i].ID == id {
					found = &snap.Issues[i]
					break
				}
			}
			if found == nil {
				return domain.NewError(domain.CodeNotFound, "阻断项不存在")
			}
			if found.Status == domain.IssueClosed {
				return domain.NewError(domain.CodeState, "已关闭阻断项不可重复复验")
			}
			if species == "" {
				species = found.SpeciesCode
			} else if species != found.SpeciesCode {
				return domain.NewError(domain.CodeInvalid, "联合复验只能选择同一物种阻断项")
			}
			selected = append(selected, found)
		}
		var obs domain.Observation
		foundObs := false
		for _, candidate := range snap.Observations {
			if candidate.ID == input.ObservationID {
				obs = candidate
				foundObs = true
				break
			}
		}
		if !foundObs {
			return domain.NewError(domain.CodeNotFound, "复验观察不存在")
		}
		if obs.SpeciesCode != species {
			return domain.NewError(domain.CodeInvalid, "复验观察与阻断物种不一致")
		}
		plan, ok := findPlan(snap.Plans, obs.SpeciesCode, obs.PlanRevision)
		if !ok || plan.LockedAt == nil {
			return domain.NewError(domain.CodeNotFound, "复验方案不存在")
		}
		result := assessment.Evaluate(snap.Plans, snap.Observations)
		for _, issue := range selected {
			passes := assessment.ObservationPassesRule(plan, obs, issue.RuleCode)
			if issue.RuleCode == "WINDOW_INCOMPLETE" || issue.RuleCode == "WINDOW_MISSING" {
				passes = !findingPresent(result, issue.SpeciesCode, issue.RuleCode)
			}
			if !passes {
				return domain.FieldError(issue.RuleCode, "复验观察尚未满足关闭条件")
			}
		}
		closed := now
		for _, issue := range selected {
			issue.Remediation = strings.TrimSpace(input.Remediation)
			issue.Status = domain.IssueClosed
			issue.VerificationObservationID = obs.ID
			issue.ClosedAt = &closed
		}
		result = assessment.Evaluate(snap.Plans, snap.Observations)
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

func findingPresent(result assessment.Result, speciesCode, ruleCode string) bool {
	for _, finding := range result.Findings {
		if finding.SpeciesCode == speciesCode && finding.RuleCode == ruleCode {
			return true
		}
	}
	return false
}
