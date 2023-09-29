/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 20:07:28
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:59
 * @FilePath: \go-distsaga\xa\resource.go
 * @Description: XA 资源接口 + 分支定义
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package xa

import (
	"context"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
)

// Context 上下文类型别名
type Context = context.Context

// Resource XA 资源接口
// 实现 XA 协议的资源必须支持 Prepare / Commit / Rollback 三阶段
type Resource interface {
	Prepare(ctx Context) error
	Commit(ctx Context) error
	Rollback(ctx Context) error
}

// Branch XA 分支定义
type Branch struct {
	Name     string        // 分支名称
	Resource Resource      // XA 资源
	Timeout  time.Duration // 超时时间
}

// NewBranch 创建 XA 分支
func NewBranch(name string, resource Resource) *Branch {
	return &Branch{
		Name:     name,
		Resource: resource,
	}
}

// WithTimeout 设置分支超时时间
func (b *Branch) WithTimeout(timeout time.Duration) *Branch {
	b.Timeout = timeout
	return b
}

// BranchState XA 分支执行状态
type BranchState struct {
	Name      string           // 分支名称
	Status    distsaga.TxState // 分支状态
	StartedAt time.Time        // 开始时间
	EndedAt   time.Time        // 结束时间
}
