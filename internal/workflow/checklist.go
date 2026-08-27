package workflow

import (
	"sort"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/domain"
)

type ChecklistItem struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Passed      bool   `json:"passed"`
	Explanation string `json:"explanation"`
}

func buildChecklist(snapshot domain.BatchSnapshot) []ChecklistItem {
	items := []ChecklistItem{
		{Code: "SOURCE_COMPLETE", Label: "来源证据完整", Passed: sourceComplete(snapshot), Explanation: "每个物种均需具备繁育地点、许可摘要、通过的检疫结论、交接时间和附件引用"},
		{Code: "PLAN_LOCKED", Label: "驯化方案已锁定", Passed: plansLocked(snapshot), Explanation: "所有计划投放物种必须配置方案，驯化开始后方案不可修订"},
		{Code: "ASSESSMENT_READY", Label: "连续观察达到适生性要求", Passed: buildViewAssessmentReady(snapshot), Explanation: "连续窗口、成活率、病害率、盐度和水位必须经过确定性计算"},
		{Code: "ISSUES_CLOSED", Label: "阻断项全部关闭", Passed: allIssuesClosed(snapshot.Issues), Explanation: "自动生成的阻断项必须记录整改措施，并在需要时关联合格复验观察"},
		{Code: "DUTY_SEPARATION", Label: "等待独立复核", Passed: snapshot.Batch.SubmittedBy != "", Explanation: "批准人必须与最近观察提交人不同"},
	}
	if snapshot.Batch.Status == domain.StatusApproved {
		items[4].Passed = snapshot.Credential != nil
		items[4].Label = "独立复核已完成"
		items[4].Explanation = "批准决定、冻结清单和签名凭据已在同一版本形成"
	}
	return items
}

func sourceComplete(snapshot domain.BatchSnapshot) bool {
	if len(snapshot.Lots) == 0 {
		return false
	}
	for _, lot := range snapshot.Lots {
		if domain.ValidateEvidence(lot) != nil {
			return false
		}
	}
	return true
}

func plansLocked(snapshot domain.BatchSnapshot) bool {
	if len(snapshot.Plans) == 0 || len(snapshot.Lots) == 0 {
		return false
	}
	covered := map[string]bool{}
	for _, plan := range snapshot.Plans {
		if plan.LockedAt == nil {
			return false
		}
		covered[plan.SpeciesCode] = true
	}
	for _, lot := range snapshot.Lots {
		if !covered[lot.SpeciesCode] {
			return false
		}
	}
	return true
}

func buildViewAssessmentReady(snapshot domain.BatchSnapshot) bool {
	return assessment.Evaluate(snapshot.Plans, snapshot.Observations).Ready
}

func sortedOpenIssueCodes(issues []domain.BlockingIssue) []string {
	var codes []string
	for _, issue := range issues {
		if issue.Status != domain.IssueClosed {
			codes = append(codes, issue.RuleCode)
		}
	}
	sort.Strings(codes)
	return codes
}
