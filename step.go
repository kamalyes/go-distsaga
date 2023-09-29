/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 11:55:18
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:08:55
 * @FilePath: \go-distsaga\step.go
 * @Description: 跨模块共享类型 - StepResult 等被多个子包共用的类型
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import "time"

// StepResult 步骤执行结果
// 被 SAGA 和 TCC 子包共享使用
type StepResult struct {
	Data  map[string]string // 结果数据
	Error error             // 执行错误
}

// StepStateItem SAGA 步骤状态项（存储在 Transaction 中）
type StepStateItem struct {
	Name      string     // 步骤名称
	Status    TxState    // 步骤状态
	Result    StepResult // 执行结果
	StartedAt time.Time  // 开始时间
	EndedAt   time.Time  // 结束时间
	Retries   int        // 已重试次数
}

// TCCBranchStateItem TCC 分支状态项（存储在 Transaction 中）
type TCCBranchStateItem struct {
	Name      string     // 分支名称
	Status    TxState    // 分支状态
	TryResult StepResult // Try 阶段结果
	StartedAt time.Time  // 开始时间
	EndedAt   time.Time  // 结束时间
	Retries   int        // 已重试次数
}

// XABranchStateItem XA 分支状态项（存储在 Transaction 中）
type XABranchStateItem struct {
	Name      string    // 分支名称
	Status    TxState   // 分支状态
	StartedAt time.Time // 开始时间
	EndedAt   time.Time // 结束时间
}

// OutboxMessageStateItem Outbox 消息状态项（存储在 Transaction 中）
type OutboxMessageStateItem struct {
	ID        string    // 消息 ID
	Status    TxState   // 消息状态
	Target    string    // 消息目标
	SentAt    time.Time // 发送时间
	Confirmed bool      // 是否已确认
	Retries   int       // 已重试次数
}
