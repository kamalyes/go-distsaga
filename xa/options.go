/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 19:31:52
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:58
 * @FilePath: \go-distsaga\xa\options.go
 * @Description: XA 函数式选项
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package xa

// Option XA 函数式选项
type Option func(*Options)

// Options XA 配置
type Options struct {
	Branches []*Branch // 分支列表
}

// WithBranches 添加 XA 分支
func WithBranches(branches ...*Branch) Option {
	return func(o *Options) {
		o.Branches = append(o.Branches, branches...)
	}
}

// NewOptions 创建 XA 选项（应用所有选项函数）
func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
