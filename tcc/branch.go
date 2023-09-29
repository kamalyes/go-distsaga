/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 13:09:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:35:19
 * @FilePath: \go-distsaga\tcc\branch.go
 * @Description: TCC 分支定义 - Try / Confirm / Cancel 三阶段
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package tcc

import (
	"context"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
)

// Context 上下文类型别名，简化使用
type Context = context.Context

// TryFunc TCC Try 阶段函数
type TryFunc func(ctx Context) (distsaga.StepResult, error)

// ConfirmFunc TCC Confirm 阶段函数
type ConfirmFunc func(ctx Context, result distsaga.StepResult) error

// CancelFunc TCC Cancel 阶段函数
type CancelFunc func(ctx Context, result distsaga.StepResult) error

// Branch TCC 分支定义
type Branch struct {
	Name    string        // 分支名称
	Try     TryFunc       // Try 阶段：资源预留
	Confirm ConfirmFunc   // Confirm 阶段：确认提交
	Cancel  CancelFunc    // Cancel 阶段：取消回滚
	Timeout time.Duration // 超时时间
	Retries int           // 重试次数
}

// NewBranch 创建 TCC 分支
func NewBranch(name string, try TryFunc, confirm ConfirmFunc, cancel CancelFunc) *Branch {
	return &Branch{
		Name:    name,
		Try:     try,
		Confirm: confirm,
		Cancel:  cancel,
	}
}

// WithTimeout 设置分支超时时间
func (b *Branch) WithTimeout(timeout time.Duration) *Branch {
	b.Timeout = timeout
	return b
}

// WithRetries 设置分支重试次数
func (b *Branch) WithRetries(retries int) *Branch {
	b.Retries = retries
	return b
}

// BranchState TCC 分支执行状态
type BranchState struct {
	Name      string              // 分支名称
	Status    distsaga.TxState    // 分支状态
	TryResult distsaga.StepResult // Try 阶段结果
	StartedAt time.Time           // 开始时间
	EndedAt   time.Time           // 结束时间
	Retries   int                 // 已重试次数
}
