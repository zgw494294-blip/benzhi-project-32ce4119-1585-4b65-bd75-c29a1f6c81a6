package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"wetland-release-workbench/internal/domain"
)

func BuildManifest(snapshot domain.BatchSnapshot, frozenAt time.Time) domain.FrozenManifest {
	items := make([]domain.ManifestItem, 0, len(snapshot.Lots))
	for _, lot := range snapshot.Lots {
		items = append(items, domain.ManifestItem{SpeciesCode: lot.SpeciesCode, Quantity: lot.Quantity})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SpeciesCode < items[j].SpeciesCode })
	return domain.FrozenManifest{BatchID: snapshot.Batch.ID, BatchVersion: snapshot.Batch.Version + 1, BatchCode: snapshot.Batch.BatchCode, SiteCode: snapshot.Batch.SiteCode, EvidenceSummary: domain.EvidenceSummary(snapshot) + "；证据摘要=" + EvidenceDigest(snapshot), Items: items, FrozenAt: frozenAt.UTC()}
}

func ManifestDigest(manifest domain.FrozenManifest) string {
	normalized := struct {
		BatchID         string                `json:"batchID"`
		BatchVersion    int64                 `json:"batchVersion"`
		BatchCode       string                `json:"batchCode"`
		SiteCode        string                `json:"siteCode"`
		EvidenceSummary string                `json:"evidenceSummary"`
		Items           []domain.ManifestItem `json:"items"`
	}{manifest.BatchID, manifest.BatchVersion, manifest.BatchCode, manifest.SiteCode, manifest.EvidenceSummary, manifest.Items}
	raw, _ := json.Marshal(normalized)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
