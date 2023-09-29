/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 08:31:52
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:55:07
 * @FilePath: \go-distsaga\tcc\options.go
 * @Description: TCC 函数式选项
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package tcc

import "time"

// Option TCC 函数式选项
type Option func(*Options)

// Options TCC 配置
type Options struct {
	Branches    []*Branch      // 分支列表
	StepTimeout time.Duration  // 分支超时时间
}

// WithBranches 添加 TCC 分支
func WithBranches(branches ...*Branch) Option {
	return func(o *Options) {
		o.Branches = append(o.Branches, branches...)
	}
}

// WithStepTimeout 设置 TCC 分支超时时间
func WithStepTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.StepTimeout = timeout
	}
}

// NewOptions 创建 TCC 选项（应用所有选项函数）
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
