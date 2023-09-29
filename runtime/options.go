/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 09:07:58
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 13:07:55
 * @FilePath: \go-distsaga\runtime\options.go
 * @Description: Runtime 函数式选项 + 默认配置常量
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package runtime

import (
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// ==================== 默认配置常量 ====================

const (
	DefaultRecoveryInterval = 10 * time.Second // 默认恢复扫描间隔
	DefaultRecoveryMaxAge   = 24 * time.Hour   // 默认恢复最大事务年龄
	DefaultStepTimeout      = 30 * time.Second // 默认步骤超时时间
	DefaultTransactionTTL   = 72 * time.Hour   // 默认事务存活时间
)

// ==================== 函数式选项 ====================

// Option Runtime 函数式选项
type Option func(*Runtime)

// WithAdapter 设置存储适配器
func WithAdapter(adapter distsaga.StoreAdapter) Option {
	return func(r *Runtime) {
		if adapter != nil {
			r.adapter = adapter
		}
	}
}

// WithNotifier 设置事务通知器
func WithNotifier(notifier distsaga.TransactionNotifier) Option {
	return func(r *Runtime) {
		if notifier != nil {
			r.notifier = notifier
		}
	}
}

// WithLogger 设置日志记录器
func WithLogger(l logger.ILogger) Option {
	return func(r *Runtime) {
		if l != nil {
			r.logger = l
		}
	}
}

// WithRetry 设置重试策略
func WithRetry(r *retry.Retry) Option {
	return func(rt *Runtime) {
		if r != nil {
			rt.retry = r
		}
	}
}

// WithRecoveryInterval 设置恢复扫描间隔
func WithRecoveryInterval(interval time.Duration) Option {
	return func(r *Runtime) {
		if interval > 0 {
			r.recoveryInterval = interval
		}
	}
}

// WithRecoveryMaxAge 设置恢复最大事务年龄
func WithRecoveryMaxAge(maxAge time.Duration) Option {
	return func(r *Runtime) {
		if maxAge > 0 {
			r.recoveryMaxAge = maxAge
		}
	}
}

// WithStepTimeout 设置步骤默认超时时间
func WithStepTimeout(timeout time.Duration) Option {
	return func(r *Runtime) {
		if timeout > 0 {
			r.stepTimeout = timeout
		}
	}
}

// WithTransactionTTL 设置事务存活时间
func WithTransactionTTL(ttl time.Duration) Option {
	return func(r *Runtime) {
		if ttl > 0 {
			r.transactionTTL = ttl
		}
	}
}

// WithEnableRecovery 设置是否启用恢复
func WithEnableRecovery(enable bool) Option {
	return func(r *Runtime) {
		r.enableRecovery = enable
	}
}
