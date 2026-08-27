package workflow

import (
	"context"
	"strings"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/store"
)

type ManifestPreview struct {
	Manifest                domain.FrozenManifest `json:"manifest"`
	Digest                  string                `json:"digest"`
	ManifestDigest          string                `json:"manifestDigest"`
	ConfirmedManifestDigest string                `json:"confirmedManifestDigest"`
	Checks                  []ChecklistItem       `json:"checks"`
}

func (s *Service) ReviewManifest(ctx context.Context, batchID string) (ManifestPreview, error) {
	snap, err := s.store.Get(ctx, batchID)
	if err != nil {
		return ManifestPreview{}, err
	}
	if snap.Batch.Status == domain.StatusApproved && snap.Manifest != nil {
		digest := assessment.ManifestDigest(*snap.Manifest)
		return ManifestPreview{Manifest: *snap.Manifest, Digest: digest, ManifestDigest: digest, ConfirmedManifestDigest: digest, Checks: buildChecklist(snap)}, nil
	}
	if err := domain.EnsureStatus(snap.Batch, domain.StatusReviewReady); err != nil {
		return ManifestPreview{}, err
	}
	manifest := assessment.BuildManifest(snap, s.now())
	checks := buildChecklist(snap)
	digest := assessment.ManifestDigest(manifest)
	return ManifestPreview{Manifest: manifest, Digest: digest, ManifestDigest: digest, ConfirmedManifestDigest: digest, Checks: checks}, nil
}

func (s *Service) Review(ctx context.Context, batchID string, input domain.ReviewInput) (BatchView, error) {
	input.Meta = normalizeMeta(input.Meta)
	if err := validateMeta(input.Meta); err != nil {
		return BatchView{}, err
	}
	if input.Decision != "approve" && input.Decision != "return" {
		return BatchView{}, domain.FieldError("decision", "必须是 approve 或 return")
	}
	if input.Decision == "return" && strings.TrimSpace(input.Reason) == "" {
		return BatchView{}, domain.FieldError("reason", "退回必须说明原因")
	}
	now := s.now()
	event := domain.NewAudit(batchID, input.Meta.Actor, "review."+input.Decision, map[string]any{"reason": input.Reason, "confirmedManifestDigest": strings.TrimSpace(input.ConfirmedManifestDigest)}, now)
	snap, _, err := s.store.Update(ctx, batchID, input.Meta.ExpectedVersion, input.Meta.IdempotencyKey, store.Fingerprint(input), event, func(snap *domain.BatchSnapshot) error {
		if err := domain.EnsureStatus(snap.Batch, domain.StatusReviewReady); err != nil {
			return err
		}
		if input.Meta.Actor == snap.Batch.SubmittedBy {
			return domain.NewError(domain.CodeForbidden, "复核人与最近观察提交人必须不同")
		}
		if input.Decision == "return" {
			snap.Batch.ReviewNote = strings.TrimSpace(input.Reason)
			return domain.Transition(&snap.Batch, domain.StatusAcclimating)
		}
		if !allIssuesClosed(snap.Issues) {
			return domain.NewError(domain.CodeState, "仍有未关闭阻断项")
		}
		manifest := assessment.BuildManifest(*snap, now)
		digest := assessment.ManifestDigest(manifest)
		if strings.TrimSpace(input.ConfirmedManifestDigest) == "" {
			return domain.FieldError("confirmedManifestDigest", "批准必须确认候选冻结清单摘要")
		}
		if !strings.EqualFold(strings.TrimSpace(input.ConfirmedManifestDigest), digest) {
			return domain.NewError(domain.CodeConflict, "候选冻结清单摘要已变化，请刷新后重新确认")
		}
		result := assessment.Evaluate(snap.Plans, snap.Observations)
		if !result.Ready {
			return domain.NewError(domain.CodeState, "观察窗口或适生性结论未达到复核条件")
		}
		if err := domain.Transition(&snap.Batch, domain.StatusApproved); err != nil {
			return err
		}
		credential := s.signer.Issue(s.newID("cred"), manifest, input.Meta.Actor, now)
		snap.Manifest = &manifest
		snap.Credential = &credential
		snap.Batch.ReviewNote = ""
		return nil
	})
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}

type CredentialVerification struct {
	Valid      bool                     `json:"valid"`
	Credential domain.ReleaseCredential `json:"credential"`
	Message    string                   `json:"message"`
}

func (s *Service) VerifyCredential(ctx context.Context, id string) (CredentialVerification, error) {
	c, err := s.store.Credential(ctx, id)
	if err != nil {
		return CredentialVerification{}, err
	}
	valid := s.signer.Verify(c)
	message := "凭据签名完整，批准范围可用于现场核验"
	if !valid {
		message = "凭据签名不匹配，请勿投放"
	}
	return CredentialVerification{Valid: valid, Credential: c, Message: message}, nil
}
