package domain

type CommandMeta struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Actor           string `json:"actor"`
}

type CreateBatchInput struct {
	BatchCode          string        `json:"batchCode"`
	SiteCode           string        `json:"siteCode"`
	PlannedReleaseDate string        `json:"plannedReleaseDate"`
	Owner              string        `json:"owner"`
	Actor              string        `json:"actor"`
	IdempotencyKey     string        `json:"idempotencyKey"`
	Lots               []SeedlingLot `json:"lots"`
}

type EvidenceInput struct {
	Meta             CommandMeta `json:"meta"`
	SpeciesCode      string      `json:"speciesCode"`
	NurseryOrigin    string      `json:"nurseryOrigin"`
	PermitDigest     string      `json:"permitDigest"`
	QuarantineResult string      `json:"quarantineResult"`
	HandoverAt       string      `json:"handoverAt"`
	EvidenceRefs     []string    `json:"evidenceRefs"`
}

type PlanInput struct {
	Meta CommandMeta     `json:"meta"`
	Plan AcclimationPlan `json:"plan"`
}

type ObservationInput struct {
	Meta        CommandMeta `json:"meta"`
	Observation Observation `json:"observation"`
}

type RemediationInput struct {
	Meta          CommandMeta `json:"meta"`
	Remediation   string      `json:"remediation"`
	ObservationID string      `json:"observationID,omitempty"`
}

type JointRemediationInput struct {
	Meta          CommandMeta `json:"meta"`
	IssueIDs      []string    `json:"issueIDs"`
	Remediation   string      `json:"remediation"`
	ObservationID string      `json:"observationID"`
}

type ReviewInput struct {
	Meta                    CommandMeta `json:"meta"`
	Decision                string      `json:"decision"`
	Reason                  string      `json:"reason,omitempty"`
	ConfirmedManifestDigest string      `json:"confirmedManifestDigest,omitempty"`
}
