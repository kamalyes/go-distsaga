/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 17:05:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:51
 * @FilePath: \go-distsaga\workflow\options.go
 * @Description: Workflow 函数式选项
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package workflow

// Handler Workflow 处理函数类型
type Handler func(wf *Workflow, data []byte) error

// Option Workflow 函数式选项
type Option func(*Options)

// Options Workflow 配置
type Options struct {
	Handler Handler // 工作流处理函数
	Data    []byte  // 工作流输入数据
}

// WithHandler 设置 Workflow 处理函数
func WithHandler(handler Handler) Option {
	return func(o *Options) {
		o.Handler = handler
	}
}

// WithData 设置 Workflow 输入数据
func WithData(data []byte) Option {
	return func(o *Options) {
		o.Data = data
	}
}

// NewOptions 创建 Workflow 选项（应用所有选项函数）
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
