package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/store"
)

type Service struct {
	store  *store.Store
	signer *assessment.Signer
	now    func() time.Time
	newID  func(string) string
}

func New(s *store.Store, signer *assessment.Signer) *Service {
	return &Service{store: s, signer: signer, now: func() time.Time { return time.Now().UTC() }, newID: randomID}
}

func randomID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func normalizeMeta(meta domain.CommandMeta) domain.CommandMeta {
	meta.Actor = strings.TrimSpace(meta.Actor)
	meta.IdempotencyKey = strings.TrimSpace(meta.IdempotencyKey)
	return meta
}

func validateMeta(meta domain.CommandMeta) error {
	if meta.Actor == "" {
		return domain.FieldError("actor", "操作人不能为空")
	}
	return nil
}

func (s *Service) Get(ctx context.Context, id string) (BatchView, error) {
	snap, err := s.store.Get(ctx, id)
	if err != nil {
		return BatchView{}, err
	}
	return buildView(snap), nil
}
func (s *Service) List(ctx context.Context) ([]domain.ReleaseBatch, error) { return s.store.List(ctx) }

func (s *Service) Trend(ctx context.Context, batchID, species string) (assessment.Trend, error) {
	snap, err := s.store.Get(ctx, batchID)
	if err != nil {
		return assessment.Trend{}, err
	}
	species = strings.TrimSpace(species)
	var plan *domain.AcclimationPlan
	for i := range snap.Plans {
		if snap.Plans[i].SpeciesCode == species && snap.Plans[i].LockedAt != nil && (plan == nil || snap.Plans[i].Revision > plan.Revision) {
			plan = &snap.Plans[i]
		}
	}
	if plan == nil {
		return assessment.Trend{}, domain.NewError(domain.CodeNotFound, "物种方案不存在")
	}
	return assessment.BuildTrend(*plan, snap.Observations, s.now())
}

type BatchView struct {
	domain.BatchSnapshot
	Assessment assessment.Result `json:"assessment"`
	Checklist  []ChecklistItem   `json:"checklist"`
}

func buildView(snap domain.BatchSnapshot) BatchView {
	view := BatchView{BatchSnapshot: snap, Assessment: assessment.Evaluate(snap.Plans, snap.Observations)}
	view.Checklist = buildChecklist(snap)
	return view
}
