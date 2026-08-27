package store

import (
	"context"
	"fmt"

	"wetland-release-workbench/internal/domain"
)

type RecoverySummary struct {
	SchemaVersion   int                        `json:"schemaVersion"`
	UnfinishedCount int                        `json:"unfinishedCount"`
	ByStatus        map[domain.BatchStatus]int `json:"byStatus"`
}

// RecoverySummary 验证迁移版本，并汇总重启后仍需继续处理的批次。
// 批次事实本身不需要重放：每条命令已经在单个 SQLite 事务中原子提交。
func (s *Store) RecoverySummary(ctx context.Context) (RecoverySummary, error) {
	result := RecoverySummary{ByStatus: map[domain.BatchStatus]int{}}
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM schema_meta LIMIT 1`).Scan(&result.SchemaVersion); err != nil {
		return result, fmt.Errorf("读取 schemaVersion: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM batches WHERE status <> ? GROUP BY status`, domain.StatusApproved)
	if err != nil {
		return result, fmt.Errorf("汇总未完成批次: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status domain.BatchStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return result, err
		}
		result.ByStatus[status] = count
		result.UnfinishedCount += count
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	return result, nil
}
