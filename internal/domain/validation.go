package domain

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

func NormalizeEvidence(lot SeedlingLot, plannedReleaseDate string, now time.Time, all []SeedlingLot) (SeedlingLot, error) {
	lot.NurseryOrigin = strings.TrimSpace(lot.NurseryOrigin)
	lot.PermitDigest = strings.TrimSpace(lot.PermitDigest)
	lot.QuarantineResult = strings.TrimSpace(lot.QuarantineResult)
	lot.HandoverAt = strings.TrimSpace(lot.HandoverAt)
	refs := make([]string, 0, len(lot.EvidenceRefs))
	seen := map[string]bool{}
	for _, ref := range lot.EvidenceRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if seen[ref] {
			return lot, NewError(CodeConflict, "同一物种不可重复引用同一证据")
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	lot.EvidenceRefs = refs
	if err := ValidateEvidence(lot); err != nil {
		return lot, err
	}
	handover, err := time.Parse(time.RFC3339, lot.HandoverAt)
	if err != nil {
		return lot, FieldError("handoverAt", "必须是带时区的 RFC3339 时间")
	}
	lot.HandoverAt = handover.UTC().Format(time.RFC3339)
	if handover.After(now.UTC()) {
		return lot, FieldError("handoverAt", "不能晚于当前时间")
	}
	planned, err := time.Parse("2006-01-02", strings.TrimSpace(plannedReleaseDate))
	if err != nil {
		return lot, FieldError("plannedReleaseDate", "必须是 YYYY-MM-DD")
	}
	if handover.UTC().After(planned.Add(24*time.Hour - time.Nanosecond)) {
		return lot, FieldError("handoverAt", "不能晚于计划投放日")
	}
	for _, other := range all {
		if other.SpeciesCode == lot.SpeciesCode {
			continue
		}
		otherHandover := strings.TrimSpace(other.HandoverAt)
		if parsed, parseErr := time.Parse(time.RFC3339, otherHandover); parseErr == nil {
			otherHandover = parsed.UTC().Format(time.RFC3339)
		}
		if strings.TrimSpace(other.PermitDigest) != "" && strings.TrimSpace(other.PermitDigest) == lot.PermitDigest {
			if strings.TrimSpace(other.NurseryOrigin) != lot.NurseryOrigin || strings.TrimSpace(other.QuarantineResult) != lot.QuarantineResult || otherHandover != lot.HandoverAt {
				return lot, NewError(CodeConflict, "同一许可摘要对应的苗源事实存在冲突")
			}
		}
		for _, a := range other.EvidenceRefs {
			for _, b := range lot.EvidenceRefs {
				if strings.TrimSpace(a) == b {
					return lot, NewError(CodeConflict, "同一批次不同物种不可重复引用同一证据")
				}
			}
		}
	}
	return lot, nil
}

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,31}$`)

func ValidateNewBatch(batch ReleaseBatch, lots []SeedlingLot) error {
	if !codePattern.MatchString(batch.BatchCode) {
		return FieldError("batchCode", "必须为 3-32 位大写字母、数字、下划线或短横线")
	}
	if !codePattern.MatchString(batch.SiteCode) {
		return FieldError("siteCode", "格式不正确")
	}
	if strings.TrimSpace(batch.Owner) == "" {
		return FieldError("owner", "负责人不能为空")
	}
	if _, err := time.Parse("2006-01-02", batch.PlannedReleaseDate); err != nil {
		return FieldError("plannedReleaseDate", "必须是 YYYY-MM-DD")
	}
	if len(lots) == 0 {
		return FieldError("lots", "至少登记一个物种配额")
	}
	seen := map[string]bool{}
	for _, lot := range lots {
		if !codePattern.MatchString(lot.SpeciesCode) {
			return FieldError("speciesCode", "格式不正确")
		}
		if lot.Quantity <= 0 {
			return FieldError("quantity", "必须大于 0")
		}
		if seen[lot.SpeciesCode] {
			return NewError(CodeConflict, "同一批次的物种配额必须唯一")
		}
		seen[lot.SpeciesCode] = true
	}
	return nil
}

func ValidateEvidence(lot SeedlingLot) error {
	missing := make([]string, 0)
	if strings.TrimSpace(lot.NurseryOrigin) == "" {
		missing = append(missing, "nurseryOrigin")
	}
	if strings.TrimSpace(lot.PermitDigest) == "" {
		missing = append(missing, "permitDigest")
	}
	if lot.QuarantineResult != "passed" {
		missing = append(missing, "quarantineResult")
	}
	if _, err := time.Parse(time.RFC3339, lot.HandoverAt); err != nil {
		missing = append(missing, "handoverAt")
	}
	if len(lot.EvidenceRefs) == 0 {
		missing = append(missing, "evidenceRefs")
	}
	refSeen := map[string]bool{}
	for _, ref := range lot.EvidenceRefs {
		if strings.TrimSpace(ref) == "" {
			missing = append(missing, "evidenceRefs")
			break
		}
		key := strings.TrimSpace(ref)
		if refSeen[key] {
			return NewError(CodeConflict, "同一物种不可重复引用同一证据")
		}
		refSeen[key] = true
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return NewError(CodeEvidenceMissing, "苗源证据不完整："+strings.Join(missing, ", "))
	}
	return nil
}

func ValidatePlan(plan AcclimationPlan) error {
	if !codePattern.MatchString(plan.SpeciesCode) || !codePattern.MatchString(plan.ZoneCode) {
		return FieldError("speciesCode/zoneCode", "格式不正确")
	}
	if plan.DurationDays < 1 || plan.DurationDays > 90 {
		return FieldError("durationDays", "必须在 1 到 90 之间")
	}
	if plan.SalinityRange.Min < 0 || plan.SalinityRange.Min > plan.SalinityRange.Max {
		return FieldError("salinityRange", "范围不正确")
	}
	if plan.WaterLevelRange.Min > plan.WaterLevelRange.Max {
		return FieldError("waterLevelRange", "范围不正确")
	}
	if plan.MinimumSurvivalRate <= 0 || plan.MinimumSurvivalRate > 1 {
		return FieldError("minimumSurvivalRate", "必须在 0 到 1 之间")
	}
	if plan.MaximumDiseaseRate < 0 || plan.MaximumDiseaseRate >= 1 {
		return FieldError("maximumDiseaseRate", "必须在 0 到 1 之间")
	}
	if plan.MinimumSampleCount < 1 {
		return FieldError("minimumSampleCount", "必须大于 0")
	}
	return nil
}

func ValidateObservation(obs Observation, plan AcclimationPlan, existing []Observation) error {
	if obs.SpeciesCode != plan.SpeciesCode || obs.ZoneCode != plan.ZoneCode || obs.PlanRevision != plan.Revision {
		return NewError(CodeInvalid, "观察记录与锁定方案不匹配")
	}
	date, err := time.Parse("2006-01-02", obs.ObservedOn)
	if err != nil {
		return FieldError("observedOn", "必须是 YYYY-MM-DD")
	}
	if date.After(time.Now().UTC().Add(24 * time.Hour)) {
		return FieldError("observedOn", "不能晚于当前日期")
	}
	if obs.SampledCount < plan.MinimumSampleCount {
		return FieldError("sampledCount", "未达到方案最低抽样数")
	}
	if obs.SurvivingCount < 0 || obs.SurvivingCount > obs.SampledCount {
		return FieldError("survivingCount", "必须处于 0 到抽样数之间")
	}
	if obs.DiseasedCount < 0 || obs.DiseasedCount > obs.SampledCount {
		return FieldError("diseasedCount", "必须处于 0 到抽样数之间")
	}
	if obs.Salinity < 0 || obs.Salinity > 100 {
		return FieldError("salinity", "必须在 0 到 100 之间")
	}
	if obs.WaterLevel < -10 || obs.WaterLevel > 20 {
		return FieldError("waterLevel", "必须在 -10 到 20 之间")
	}
	if strings.TrimSpace(obs.SubmittedBy) == "" {
		return FieldError("submittedBy", "提交人不能为空")
	}
	for _, old := range existing {
		if old.SpeciesCode == obs.SpeciesCode && old.ZoneCode == obs.ZoneCode && old.ObservedOn == obs.ObservedOn {
			return NewError(CodeConflict, "同一物种分区观察日期不可重复")
		}
	}
	return nil
}
