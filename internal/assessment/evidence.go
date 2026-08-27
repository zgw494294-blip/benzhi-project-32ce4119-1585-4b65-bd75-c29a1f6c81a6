package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"wetland-release-workbench/internal/domain"
)

// EvidenceDigest 将批准所依据的来源、方案、观察和整改事实规范化后摘要。
// 审计事件不参与摘要，避免日志展示字段影响已冻结的业务事实。
func EvidenceDigest(snapshot domain.BatchSnapshot) string {
	facts := evidenceFacts{
		Lots:         append([]domain.SeedlingLot(nil), snapshot.Lots...),
		Plans:        append([]domain.AcclimationPlan(nil), snapshot.Plans...),
		Observations: append([]domain.Observation(nil), snapshot.Observations...),
		Issues:       append([]domain.BlockingIssue(nil), snapshot.Issues...),
	}
	sort.Slice(facts.Lots, func(i, j int) bool {
		return facts.Lots[i].SpeciesCode < facts.Lots[j].SpeciesCode
	})
	sort.Slice(facts.Plans, func(i, j int) bool {
		if facts.Plans[i].SpeciesCode == facts.Plans[j].SpeciesCode {
			return facts.Plans[i].Revision < facts.Plans[j].Revision
		}
		return facts.Plans[i].SpeciesCode < facts.Plans[j].SpeciesCode
	})
	sort.Slice(facts.Observations, func(i, j int) bool {
		if facts.Observations[i].ObservedOn == facts.Observations[j].ObservedOn {
			return facts.Observations[i].ID < facts.Observations[j].ID
		}
		return facts.Observations[i].ObservedOn < facts.Observations[j].ObservedOn
	})
	sort.Slice(facts.Issues, func(i, j int) bool {
		if facts.Issues[i].SpeciesCode == facts.Issues[j].SpeciesCode {
			return facts.Issues[i].RuleCode < facts.Issues[j].RuleCode
		}
		return facts.Issues[i].SpeciesCode < facts.Issues[j].SpeciesCode
	})
	raw, _ := json.Marshal(facts)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type evidenceFacts struct {
	Lots         []domain.SeedlingLot     `json:"lots"`
	Plans        []domain.AcclimationPlan `json:"plans"`
	Observations []domain.Observation     `json:"observations"`
	Issues       []domain.BlockingIssue   `json:"issues"`
}
