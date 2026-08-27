package domain

import "time"

func NewAudit(batchID, actor, action string, details map[string]any, now time.Time) AuditEvent {
	if details == nil {
		details = map[string]any{}
	}
	return AuditEvent{BatchID: batchID, Actor: actor, Action: action, Details: details, CreatedAt: now.UTC()}
}

func EvidenceSummary(snapshot BatchSnapshot) string {
	return "苗源证据=" + itoa(len(snapshot.Lots)) + "；方案=" + itoa(len(snapshot.Plans)) + "；观察=" + itoa(len(snapshot.Observations)) + "；已关闭阻断=" + itoa(closedCount(snapshot.Issues))
}

func closedCount(issues []BlockingIssue) int {
	n := 0
	for _, issue := range issues {
		if issue.Status == IssueClosed {
			n++
		}
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := [20]byte{}
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
