/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 13:52:38
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 09:31:28
 * @FilePath: \go-distsaga\outbox\options.go
 * @Description: Outbox 函数式选项
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package outbox

import "context"

// Option Outbox 函数式选项
type Option func(*Options)

// Options Outbox 配置
type Options struct {
	Messages   []*Message         // 消息列表
	BusinessOp BusinessOp         // 业务操作函数
}

// WithMessage 添加 Outbox 消息
func WithMessage(msg *Message) Option {
	return func(o *Options) {
		o.Messages = append(o.Messages, msg)
	}
}

// WithMessages 批量添加 Outbox 消息
func WithMessages(messages ...*Message) Option {
	return func(o *Options) {
		o.Messages = append(o.Messages, messages...)
	}
}

// WithBusinessOp 设置 Outbox 业务操作
func WithBusinessOp(op func(ctx context.Context) error) Option {
	return func(o *Options) {
		o.BusinessOp = op
	}
}

// NewOptions 创建 Outbox 选项（应用所有选项函数）
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
