package domain

import "time"

type BatchStatus string

const (
	StatusDraft            BatchStatus = "draft"
	StatusEvidenceReady    BatchStatus = "evidence_ready"
	StatusAcclimating      BatchStatus = "acclimating"
	StatusNeedsRemediation BatchStatus = "needs_remediation"
	StatusReviewReady      BatchStatus = "review_ready"
	StatusApproved         BatchStatus = "approved"
)

type ReleaseBatch struct {
	ID                 string      `json:"id"`
	BatchCode          string      `json:"batchCode"`
	SiteCode           string      `json:"siteCode"`
	PlannedReleaseDate string      `json:"plannedReleaseDate"`
	Owner              string      `json:"owner"`
	Status             BatchStatus `json:"status"`
	Version            int64       `json:"version"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`
	SubmittedBy        string      `json:"submittedBy,omitempty"`
	ReviewNote         string      `json:"reviewNote,omitempty"`
}

type SeedlingLot struct {
	ID               string   `json:"id"`
	BatchID          string   `json:"batchID"`
	SpeciesCode      string   `json:"speciesCode"`
	Quantity         int      `json:"quantity"`
	NurseryOrigin    string   `json:"nurseryOrigin"`
	PermitDigest     string   `json:"permitDigest"`
	QuarantineResult string   `json:"quarantineResult"`
	HandoverAt       string   `json:"handoverAt"`
	EvidenceRefs     []string `json:"evidenceRefs"`
}

type NumberRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type AcclimationPlan struct {
	ID                  string      `json:"id"`
	BatchID             string      `json:"batchID"`
	SpeciesCode         string      `json:"speciesCode"`
	Revision            int         `json:"revision"`
	ZoneCode            string      `json:"zoneCode"`
	DurationDays        int         `json:"durationDays"`
	SalinityRange       NumberRange `json:"salinityRange"`
	WaterLevelRange     NumberRange `json:"waterLevelRange"`
	MinimumSurvivalRate float64     `json:"minimumSurvivalRate"`
	MaximumDiseaseRate  float64     `json:"maximumDiseaseRate"`
	MinimumSampleCount  int         `json:"minimumSampleCount"`
	LockedAt            *time.Time  `json:"lockedAt,omitempty"`
}

type Observation struct {
	ID             string    `json:"id"`
	BatchID        string    `json:"batchID"`
	SpeciesCode    string    `json:"speciesCode"`
	PlanRevision   int       `json:"planRevision"`
	ObservedOn     string    `json:"observedOn"`
	ZoneCode       string    `json:"zoneCode"`
	SampledCount   int       `json:"sampledCount"`
	SurvivingCount int       `json:"survivingCount"`
	DiseasedCount  int       `json:"diseasedCount"`
	Salinity       float64   `json:"salinity"`
	WaterLevel     float64   `json:"waterLevel"`
	Notes          string    `json:"notes"`
	SubmittedBy    string    `json:"submittedBy"`
	CreatedAt      time.Time `json:"createdAt"`
}

type IssueStatus string

const (
	IssueOpen       IssueStatus = "open"
	IssueRemediated IssueStatus = "remediated"
	IssueClosed     IssueStatus = "closed"
)

type BlockingIssue struct {
	ID                        string      `json:"id"`
	BatchID                   string      `json:"batchID"`
	SpeciesCode               string      `json:"speciesCode"`
	RuleCode                  string      `json:"ruleCode"`
	Severity                  string      `json:"severity"`
	Description               string      `json:"description"`
	DueDate                   string      `json:"dueDate"`
	Status                    IssueStatus `json:"status"`
	Remediation               string      `json:"remediation,omitempty"`
	VerificationObservationID string      `json:"verificationObservationID,omitempty"`
	ClosedAt                  *time.Time  `json:"closedAt,omitempty"`
}

type EvidenceRecord struct {
	BatchID         string      `json:"batchID"`
	SpeciesCode     string      `json:"speciesCode"`
	BatchCode       string      `json:"batchCode"`
	Owner           string      `json:"owner"`
	SpeciesQuantity int         `json:"speciesQuantity"`
	Severity        string      `json:"severity"`
	DueDate         string      `json:"dueDate"`
	Timing          string      `json:"timing"`
	Status          IssueStatus `json:"status"`
	IssueID         string      `json:"issueID"`
}

type RemediationQueueItem = EvidenceRecord

type RemediationQueue struct {
	Items  []EvidenceRecord `json:"items"`
	Counts map[string]int   `json:"counts"`
}

type ManifestItem struct {
	SpeciesCode string `json:"speciesCode"`
	Quantity    int    `json:"quantity"`
}

type FrozenManifest struct {
	BatchID         string         `json:"batchID"`
	BatchVersion    int64          `json:"batchVersion"`
	BatchCode       string         `json:"batchCode"`
	SiteCode        string         `json:"siteCode"`
	EvidenceSummary string         `json:"evidenceSummary"`
	Items           []ManifestItem `json:"items"`
	FrozenAt        time.Time      `json:"frozenAt"`
}

type ReleaseCredential struct {
	ID                string         `json:"id"`
	BatchID           string         `json:"batchID"`
	ManifestDigest    string         `json:"manifestDigest"`
	SiteCode          string         `json:"siteCode"`
	SpeciesQuantities []ManifestItem `json:"speciesQuantities"`
	ApprovedAt        time.Time      `json:"approvedAt"`
	ApprovedBy        string         `json:"approvedBy"`
	Signature         string         `json:"signature"`
	KeyID             string         `json:"keyID"`
}

type AuditEvent struct {
	ID        int64          `json:"id"`
	BatchID   string         `json:"batchID"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"createdAt"`
}

type BatchSnapshot struct {
	Batch        ReleaseBatch       `json:"batch"`
	Lots         []SeedlingLot      `json:"lots"`
	Plans        []AcclimationPlan  `json:"plans"`
	Observations []Observation      `json:"observations"`
	Issues       []BlockingIssue    `json:"issues"`
	Manifest     *FrozenManifest    `json:"manifest,omitempty"`
	Credential   *ReleaseCredential `json:"credential,omitempty"`
	Audit        []AuditEvent       `json:"audit"`
}
