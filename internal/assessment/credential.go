package assessment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"wetland-release-workbench/internal/domain"
)

type Signer struct {
	keyID string
	key   []byte
}

func NewSigner(keyID string, key []byte) *Signer {
	return &Signer{keyID: keyID, key: append([]byte(nil), key...)}
}

func (s *Signer) Issue(id string, manifest domain.FrozenManifest, approvedBy string, approvedAt time.Time) domain.ReleaseCredential {
	c := domain.ReleaseCredential{ID: id, BatchID: manifest.BatchID, ManifestDigest: ManifestDigest(manifest), SiteCode: manifest.SiteCode, SpeciesQuantities: append([]domain.ManifestItem(nil), manifest.Items...), ApprovedAt: approvedAt.UTC(), ApprovedBy: approvedBy, KeyID: s.keyID}
	c.Signature = s.sign(c)
	return c
}

func (s *Signer) Verify(c domain.ReleaseCredential) bool {
	if c.KeyID != s.keyID || c.Signature == "" {
		return false
	}
	expected := s.sign(c)
	return hmac.Equal([]byte(expected), []byte(c.Signature))
}

func (s *Signer) sign(c domain.ReleaseCredential) string {
	payload := struct {
		ID                string                `json:"id"`
		BatchID           string                `json:"batchID"`
		ManifestDigest    string                `json:"manifestDigest"`
		SiteCode          string                `json:"siteCode"`
		SpeciesQuantities []domain.ManifestItem `json:"speciesQuantities"`
		ApprovedAt        string                `json:"approvedAt"`
		ApprovedBy        string                `json:"approvedBy"`
		KeyID             string                `json:"keyID"`
	}{c.ID, c.BatchID, strings.ToLower(c.ManifestDigest), c.SiteCode, c.SpeciesQuantities, c.ApprovedAt.UTC().Format(time.RFC3339Nano), c.ApprovedBy, c.KeyID}
	raw, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
