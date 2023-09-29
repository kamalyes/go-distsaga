/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 13:28:55
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 18:07:22
 * @FilePath: \go-distsaga\saga\options.go
 * @Description: SAGA 函数式选项
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package saga

import "time"

// Option SAGA 函数式选项
type Option func(*Options)

// Options SAGA 配置
type Options struct {
	Steps       []*Step       // 步骤列表
	StepTimeout time.Duration // 步骤超时时间
}

// WithSteps 添加 SAGA 步骤
func WithSteps(steps ...*Step) Option {
	return func(o *Options) {
		o.Steps = append(o.Steps, steps...)
	}
}

// WithStepTimeout 设置 SAGA 步骤超时时间
func WithStepTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.StepTimeout = timeout
	}
}

// NewOptions 创建 SAGA 选项（应用所有选项函数）
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
