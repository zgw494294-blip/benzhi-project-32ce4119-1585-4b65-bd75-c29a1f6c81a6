package assessment

import "wetland-release-workbench/internal/domain"

type SpeciesMetrics struct {
	SpeciesCode     string  `json:"speciesCode"`
	ObservationDays int     `json:"observationDays"`
	WindowDays      int     `json:"windowDays"`
	Sampled         int     `json:"sampled"`
	Surviving       int     `json:"surviving"`
	Diseased        int     `json:"diseased"`
	SurvivalRate    float64 `json:"survivalRate"`
	DiseaseRate     float64 `json:"diseaseRate"`
	Passed          bool    `json:"passed"`
}

type Finding struct {
	SpeciesCode string `json:"speciesCode"`
	RuleCode    string `json:"ruleCode"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	DueDays     int    `json:"dueDays"`
}

type Result struct {
	Ready    bool             `json:"ready"`
	Metrics  []SpeciesMetrics `json:"metrics"`
	Findings []Finding        `json:"findings"`
}

func FindingToIssue(f Finding, batchID, id, dueDate string) domain.BlockingIssue {
	return domain.BlockingIssue{ID: id, BatchID: batchID, SpeciesCode: f.SpeciesCode, RuleCode: f.RuleCode, Severity: f.Severity, Description: f.Description, DueDate: dueDate, Status: domain.IssueOpen}
}
