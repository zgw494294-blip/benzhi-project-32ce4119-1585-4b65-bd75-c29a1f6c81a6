package domain

func CanTransition(from, to BatchStatus) bool {
	allowed := map[BatchStatus]map[BatchStatus]bool{
		StatusDraft:            {StatusEvidenceReady: true},
		StatusEvidenceReady:    {StatusAcclimating: true},
		StatusAcclimating:      {StatusNeedsRemediation: true, StatusReviewReady: true},
		StatusNeedsRemediation: {StatusReviewReady: true},
		StatusReviewReady:      {StatusAcclimating: true, StatusApproved: true},
	}
	return allowed[from][to]
}

func Transition(batch *ReleaseBatch, to BatchStatus) error {
	if batch.Status == StatusApproved {
		return NewError(CodeFrozen, "已批准批次不可再修改")
	}
	if !CanTransition(batch.Status, to) {
		return NewError(CodeState, "不允许的状态迁移："+string(batch.Status)+" -> "+string(to))
	}
	batch.Status = to
	return nil
}

func EnsureMutable(batch ReleaseBatch) error {
	if batch.Status == StatusApproved {
		return NewError(CodeFrozen, "冻结清单后不能修改事实")
	}
	return nil
}

func EnsureStatus(batch ReleaseBatch, statuses ...BatchStatus) error {
	for _, status := range statuses {
		if batch.Status == status {
			return nil
		}
	}
	return NewError(CodeState, "当前状态不允许执行此操作："+string(batch.Status))
}
