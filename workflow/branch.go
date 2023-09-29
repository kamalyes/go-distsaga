/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 13:35:17
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:22
 * @FilePath: \go-distsaga\workflow\branch.go
 * @Description: Workflow 分支定义 - 灵活编排
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package workflow

import (
	"context"
	"sync"

	"github.com/kamalyes/go-logger"
)

// Context 上下文类型别名
type Context = context.Context

// Branch Workflow 分支定义
type Branch struct {
	Name       string                               // 分支名称
	OnConfirm  func(ctx Context, data []byte) error // 确认回调
	OnCancel   func(ctx Context, data []byte) error // 取消回调
	OnRollback func(ctx Context, data []byte) error // 回滚回调
}

// Workflow 工作流上下文
type Workflow struct {
	ID       string         // 工作流 ID
	Context  Context        // 工作流上下文
	branches []*Branch      // 已注册的分支列表
	mu       sync.Mutex     // 保护 branches 的并发安全
	logger   logger.ILogger // 日志记录器
}

// NewBranch 创建新的 Workflow 分支
func (wf *Workflow) NewBranch() *Branch {
	branch := &Branch{}
	wf.mu.Lock()
	wf.branches = append(wf.branches, branch)
	wf.mu.Unlock()
	return branch
}

// OnRollback 注册回滚回调
func (wf *Workflow) OnRollback(fn func(ctx Context, data []byte) error) *Branch {
	branch := wf.NewBranch()
	branch.OnRollback = fn
	return branch
}

// OnConfirm 注册确认回调
func (wf *Workflow) OnConfirm(fn func(ctx Context, data []byte) error) *Branch {
	branch := wf.NewBranch()
	branch.OnConfirm = fn
	return branch
}

// OnCancel 注册取消回调
func (wf *Workflow) OnCancel(fn func(ctx Context, data []byte) error) *Branch {
	branch := wf.NewBranch()
	branch.OnCancel = fn
	return branch
}

// GetBranches 获取已注册的分支列表
func (wf *Workflow) GetBranches() []*Branch {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	result := make([]*Branch, len(wf.branches))
	copy(result, wf.branches)
	return result
}
