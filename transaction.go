/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 10:53:28
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:58:51
 * @FilePath: \go-distsaga\transaction.go
 * @Description: Transaction 核心数据模型 - 事务生命周期管理
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"fmt"
	"time"

	"github.com/kamalyes/go-toolbox/pkg/idgen"
)

// Transaction 分布式事务核心数据模型
// 记录事务的完整生命周期，包括 ID、名称、模式、状态、各模式分支状态等
type Transaction struct {
	ID          string                   `json:"id"`                        // 事务 ID（基于 Snowflake 生成）
	Name        string                   `json:"name"`                      // 事务名称
	Mode        TxMode                   `json:"mode"`                      // 事务模式（SAGA/TCC/XA/Workflow/Outbox）
	State       TxState                  `json:"state"`                     // 当前状态
	Steps       []StepStateItem          `json:"steps,omitempty"`           // SAGA 步骤状态列表
	TCCBranches []TCCBranchStateItem     `json:"tcc_branches,omitempty"`    // TCC 分支状态列表
	XABranches  []XABranchStateItem      `json:"xa_branches,omitempty"`     // XA 分支状态列表
	OutboxMsgs  []OutboxMessageStateItem `json:"outbox_messages,omitempty"` // Outbox 消息状态列表
	Payload     map[string]interface{}   `json:"payload,omitempty"`         // 附加数据
	CreatedAt   time.Time                `json:"created_at"`                // 创建时间
	UpdatedAt   time.Time                `json:"updated_at"`                // 更新时间
	FinishedAt  *time.Time               `json:"finished_at,omitempty"`     // 结束时间（终态时设置）
}

// NewTransaction 创建新事务
// 自动生成基于 Snowflake 的事务 ID，初始状态为 PENDING
func NewTransaction(name string, mode TxMode) *Transaction {
	now := time.Now()
	return &Transaction{
		ID:        generateTransactionID(),
		Name:      name,
		Mode:      mode,
		State:     StatePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TransitionTo 执行状态转换
// 校验状态转换是否合法（基于当前事务模式的状态流转表），不合法则返回 ErrStateTransition
func (tx *Transaction) TransitionTo(target TxState) error {
	if !CanTransition(tx.Mode, tx.State, target) {
		return fmt.Errorf("%w: %s -> %s (mode=%s)", ErrStateTransition, tx.State, target, tx.Mode)
	}
	tx.State = target
	tx.UpdatedAt = time.Now()
	if target.IsFinal() {
		now := time.Now()
		tx.FinishedAt = &now
	}
	return nil
}

// IsFinal 判断事务是否处于终态
func (tx *Transaction) IsFinal() bool {
	return tx.State.IsFinal()
}

// GetStepResult 获取指定步骤的执行结果（SAGA 模式）
func (tx *Transaction) GetStepResult(stepName string) (StepResult, bool) {
	for i := range tx.Steps {
		if tx.Steps[i].Name == stepName {
			return tx.Steps[i].Result, true
		}
	}
	return StepResult{}, false
}

// GetTCCBranchResult 获取指定 TCC 分支的 Try 阶段结果
func (tx *Transaction) GetTCCBranchResult(branchName string) (StepResult, bool) {
	for i := range tx.TCCBranches {
		if tx.TCCBranches[i].Name == branchName {
			return tx.TCCBranches[i].TryResult, true
		}
	}
	return StepResult{}, false
}

// defaultIDGenerator 默认 ID 生成器（基于 Snowflake 算法）
var defaultIDGenerator = idgen.NewSnowflakeGenerator(1, 1)

// generateTransactionID 生成事务 ID
// 格式：distsaga-{SnowflakeID}
func generateTransactionID() string {
	return fmt.Sprintf("distsaga-%s", defaultIDGenerator.GenerateTraceID())
}
