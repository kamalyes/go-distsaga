/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 09:37:05
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 21:05:33
 * @FilePath: \go-distsaga\state.go
 * @Description: 事务状态枚举 + 状态机 - 定义各模式的状态流转规则
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"fmt"
)

// ==================== 事务状态定义 ====================

// TxState 事务状态
type TxState int

const (
	StatePending      TxState = iota // 待执行
	StateRunning                      // 执行中（SAGA/Workflow）
	StateTrying                       // Try 阶段（TCC）
	StateConfirming                   // Confirm 阶段（TCC）
	StateCanceling                    // Cancel 阶段（TCC）
	StateCompensating                 // 补偿中（SAGA/Workflow）
	StatePreparing                    // Prepare 阶段（XA）
	StatePrepared                     // 已 Prepare（XA）
	StateCommitting                   // 提交中（XA/Outbox）
	StateAborting                     // 中止中（XA）
	StateRollbacking                  // 回滚中
	StateSubmitting                   // 提交业务操作中（Outbox）
	StateSubmitted                    // 业务操作已提交（Outbox）
	StateCommitted                    // 已提交（终态）
	StateCompensated                  // 已补偿（终态）
	StateRolledback                   // 已回滚（终态）
	StateFailed                       // 失败（终态）
	StateSuspended                    // 挂起（需人工干预）
)

// String 返回事务状态的字符串表示
func (s TxState) String() string {
	names := map[TxState]string{
		StatePending:      "PENDING",
		StateRunning:      "RUNNING",
		StateTrying:       "TRYING",
		StateConfirming:   "CONFIRMING",
		StateCanceling:    "CANCELING",
		StateCompensating: "COMPENSATING",
		StatePreparing:    "PREPARING",
		StatePrepared:     "PREPARED",
		StateCommitting:   "COMMITTING",
		StateAborting:     "ABORTING",
		StateRollbacking:  "ROLLBACKING",
		StateSubmitting:   "SUBMITTING",
		StateSubmitted:    "SUBMITTED",
		StateCommitted:    "COMMITTED",
		StateCompensated:  "COMPENSATED",
		StateRolledback:   "ROLLEDBACK",
		StateFailed:       "FAILED",
		StateSuspended:    "SUSPENDED",
	}
	if name, ok := names[s]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", s)
}

// IsFinal 判断事务状态是否为终态
// 终态包括：COMMITTED / COMPENSATED / ROLLEDBACK / FAILED
func (s TxState) IsFinal() bool {
	switch s {
	case StateCommitted, StateCompensated, StateRolledback, StateFailed:
		return true
	default:
		return false
	}
}

// ==================== 事务模式定义 ====================

// TxMode 事务模式
type TxMode int

const (
	ModeSaga     TxMode = iota // SAGA 模式：正向执行 + 逆序补偿
	ModeTCC                    // TCC 模式：Try / Confirm / Cancel
	ModeXA                     // XA 模式：Prepare / Commit / Rollback
	ModeWorkflow               // Workflow 模式：灵活编排，可混合 SAGA/TCC/XA
	ModeOutbox                 // Outbox 模式：两阶段消息（Better Outbox）
)

// String 返回事务模式的字符串表示
func (m TxMode) String() string {
	names := map[TxMode]string{
		ModeSaga:     "SAGA",
		ModeTCC:      "TCC",
		ModeXA:       "XA",
		ModeWorkflow: "WORKFLOW",
		ModeOutbox:   "OUTBOX",
	}
	if name, ok := names[m]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", m)
}

// ==================== 各模式状态流转规则 ====================

// sagaTransitions SAGA 模式状态流转表
// PENDING → RUNNING → COMMITTED / COMPENSATING → COMPENSATED
var sagaTransitions = map[TxState][]TxState{
	StatePending:      {StateRunning},
	StateRunning:      {StateCommitted, StateCompensating, StateFailed, StateSuspended},
	StateCompensating: {StateCompensated, StateFailed, StateSuspended},
}

// tccTransitions TCC 模式状态流转表
// PENDING → TRYING → CONFIRMING → COMMITTED / CANCELING → ROLLEDBACK
var tccTransitions = map[TxState][]TxState{
	StatePending:    {StateTrying},
	StateTrying:     {StateConfirming, StateCanceling, StateFailed, StateSuspended},
	StateConfirming: {StateCommitted, StateFailed, StateSuspended},
	StateCanceling:  {StateRolledback, StateFailed, StateSuspended},
}

// xaTransitions XA 模式状态流转表
// PENDING → PREPARING → PREPARED → COMMITTING → COMMITTED / ABORTING → ROLLEDBACK
var xaTransitions = map[TxState][]TxState{
	StatePending:    {StatePreparing},
	StatePreparing:  {StatePrepared, StateAborting, StateFailed, StateSuspended},
	StatePrepared:   {StateCommitting, StateAborting},
	StateCommitting: {StateCommitted, StateFailed, StateSuspended},
	StateAborting:   {StateRolledback, StateFailed, StateSuspended},
}

// outboxTransitions Outbox 模式状态流转表
// PENDING → SUBMITTING → SUBMITTED → COMMITTED
var outboxTransitions = map[TxState][]TxState{
	StatePending:    {StateSubmitting},
	StateSubmitting: {StateSubmitted, StateFailed, StateSuspended},
	StateSubmitted:  {StateCommitted},
}

// getTransitions 根据事务模式获取对应的状态流转表
func getTransitions(mode TxMode) map[TxState][]TxState {
	switch mode {
	case ModeSaga:
		return sagaTransitions
	case ModeTCC:
		return tccTransitions
	case ModeXA:
		return xaTransitions
	case ModeOutbox:
		return outboxTransitions
	case ModeWorkflow:
		return sagaTransitions // Workflow 复用 SAGA 的状态流转
	default:
		return sagaTransitions
	}
}

// CanTransition 判断在指定模式下，从 from 状态到 to 状态的转换是否合法
func CanTransition(mode TxMode, from, to TxState) bool {
	transitions := getTransitions(mode)
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
