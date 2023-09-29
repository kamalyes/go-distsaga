/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 15:51:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 19:28:51
 * @FilePath: \go-distsaga\saga\step.go
 * @Description: SAGA 步骤定义 - 正向动作 + 补偿动作
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package saga

import (
	"context"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
)

// Context 上下文类型别名，简化使用
type Context = context.Context

// ActionFunc 正向动作函数
type ActionFunc func(ctx Context) (distsaga.StepResult, error)

// CompensateFunc 补偿动作函数
type CompensateFunc func(ctx Context, result distsaga.StepResult) error

// Step SAGA 步骤定义
// 每个步骤包含正向动作和补偿动作，补偿在正向失败时逆序执行
type Step struct {
	Name       string         // 步骤名称
	Action     ActionFunc     // 正向动作
	Compensate CompensateFunc // 补偿动作
	Timeout    time.Duration  // 超时时间
	Retries    int            // 重试次数
}

// NewStep 创建 SAGA 步骤
func NewStep(name string, action ActionFunc, compensate CompensateFunc) *Step {
	return &Step{
		Name:       name,
		Action:     action,
		Compensate: compensate,
	}
}

// WithTimeout 设置步骤超时时间
func (s *Step) WithTimeout(timeout time.Duration) *Step {
	s.Timeout = timeout
	return s
}

// WithRetries 设置步骤重试次数
func (s *Step) WithRetries(retries int) *Step {
	s.Retries = retries
	return s
}

// StepState 步骤执行状态
type StepState struct {
	Name      string              // 步骤名称
	Status    distsaga.TxState    // 步骤状态
	Result    distsaga.StepResult // 执行结果
	StartedAt time.Time           // 开始时间
	EndedAt   time.Time           // 结束时间
	Retries   int                 // 已重试次数
}
