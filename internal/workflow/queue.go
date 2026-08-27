package workflow

import (
	"context"
	"sort"
	"strings"
	"time"

	"wetland-release-workbench/internal/domain"
)

type QueueFilters struct {
	Owner       string
	SpeciesCode string
	Severity    string
	Timing      string
	DueDateFrom string
	DueDateTo   string
}

func (s *Service) RemediationQueue(ctx context.Context, filters QueueFilters) (domain.RemediationQueue, error) {
	filters.Owner = strings.TrimSpace(filters.Owner)
	filters.SpeciesCode = strings.TrimSpace(filters.SpeciesCode)
	if filters.Severity != "" && filters.Severity != "critical" && filters.Severity != "high" && filters.Severity != "medium" {
		return domain.RemediationQueue{}, domain.FieldError("severity", "枚举值无效")
	}
	if filters.Timing != "" && filters.Timing != "overdue" && filters.Timing != "today" && filters.Timing != "due_soon" && filters.Timing != "normal" {
		return domain.RemediationQueue{}, domain.FieldError("timing", "枚举值无效")
	}
	var from, to time.Time
	var err error
	if filters.DueDateFrom != "" {
		from, err = time.Parse("2006-01-02", filters.DueDateFrom)
		if err != nil {
			return domain.RemediationQueue{}, domain.FieldError("dueDateFrom", "必须是 YYYY-MM-DD")
		}
	}
	if filters.DueDateTo != "" {
		to, err = time.Parse("2006-01-02", filters.DueDateTo)
		if err != nil {
			return domain.RemediationQueue{}, domain.FieldError("dueDateTo", "必须是 YYYY-MM-DD")
		}
		if !from.IsZero() && to.Before(from) {
			return domain.RemediationQueue{}, domain.FieldError("dueDateTo", "不能早于起始日期")
		}
	}
	items, err := s.store.RemediationItems(ctx)
	if err != nil {
		return domain.RemediationQueue{}, err
	}
	today := s.now().UTC().Truncate(24 * time.Hour)
	result := domain.RemediationQueue{Counts: map[string]int{"overdue": 0, "today": 0, "due_soon": 0, "normal": 0}}
	for _, item := range items {
		if filters.Owner != "" && item.Owner != filters.Owner || filters.SpeciesCode != "" && item.SpeciesCode != filters.SpeciesCode || filters.Severity != "" && item.Severity != filters.Severity {
			continue
		}
		due, parseErr := time.Parse("2006-01-02", item.DueDate)
		if parseErr != nil {
			return domain.RemediationQueue{}, domain.FieldError("dueDate", "持久化期限无效")
		}
		days := int(due.Sub(today).Hours() / 24)
		switch {
		case days < 0:
			item.Timing = "overdue"
		case days == 0:
			item.Timing = "today"
		case days <= 3:
			item.Timing = "due_soon"
		default:
			item.Timing = "normal"
		}
		if filters.Timing != "" && item.Timing != filters.Timing || !from.IsZero() && due.Before(from) || !to.IsZero() && due.After(to) {
			continue
		}
		result.Items = append(result.Items, item)
		result.Counts[item.Timing]++
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		rank := map[string]int{"overdue": 0, "today": 1, "due_soon": 2, "normal": 3}
		if rank[a.Timing] != rank[b.Timing] {
			return rank[a.Timing] < rank[b.Timing]
		}
		sr := map[string]int{"critical": 0, "high": 1, "medium": 2}
		if sr[a.Severity] != sr[b.Severity] {
			return sr[a.Severity] < sr[b.Severity]
		}
		if a.DueDate != b.DueDate {
			return a.DueDate < b.DueDate
		}
		if a.BatchCode != b.BatchCode {
			return a.BatchCode < b.BatchCode
		}
		return a.IssueID < b.IssueID
	})
	return result, nil
}

func (s *Service) QueryRemediationQueue(ctx context.Context, filters QueueFilters) (domain.RemediationQueue, error) {
	return s.RemediationQueue(ctx, filters)
}
