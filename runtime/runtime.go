/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 10:35:11
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 15:19:33
 * @FilePath: \go-distsaga\runtime\runtime.go
 * @Description: Runtime - 分布式事务运行时引擎，核心结构体定义
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package runtime

import (
	"context"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// Runtime 分布式事务运行时引擎
// 统一管理所有事务模式的执行、通知和恢复
type Runtime struct {
	adapter          distsaga.StoreAdapter        // 存储适配器
	notifier         distsaga.TransactionNotifier // 事务通知器
	logger           logger.ILogger               // 日志记录器
	retry            *retry.Retry                 // 重试策略
	recovery         *distsaga.Recovery           // 崩溃恢复器
	recoveryInterval time.Duration                // 恢复扫描间隔
	recoveryMaxAge   time.Duration                // 恢复最大事务年龄
	stepTimeout      time.Duration                // 步骤默认超时
	transactionTTL   time.Duration                // 事务存活时间
	enableRecovery   bool                         // 是否启用恢复
}

// New 创建分布式事务运行时
func New(opts ...Option) (*Runtime, error) {
	r := &Runtime{
		adapter:          distsaga.NewMemoryAdapter(),
		logger:           distsaga.NewDefaultLogger(),
		recoveryInterval: DefaultRecoveryInterval,
		recoveryMaxAge:   DefaultRecoveryMaxAge,
		stepTimeout:      DefaultStepTimeout,
		transactionTTL:   DefaultTransactionTTL,
		enableRecovery:   true,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.adapter == nil {
		return nil, distsaga.ErrInvalidAdapter
	}

	if r.retry == nil {
		r.retry = retry.NewRetry().
			SetAttemptCount(3).
			SetInterval(100 * time.Millisecond)
	}

	if r.enableRecovery {
		r.recovery = distsaga.NewRecovery(r.adapter, r.logger, r.recoveryInterval, r.recoveryMaxAge)
	}

	r.logger.Infof("Distsaga Runtime initialized (adapter=%T, recovery=%v)", r.adapter, r.enableRecovery)
	return r, nil
}

// StartRecovery 启动崩溃恢复
func (r *Runtime) StartRecovery(ctx context.Context) error {
	if r.recovery == nil {
		return distsaga.ErrRecoveryFailed
	}
	return r.recovery.Start(ctx)
}

// StopRecovery 停止崩溃恢复
func (r *Runtime) StopRecovery() {
	if r.recovery != nil {
		r.recovery.Stop()
	}
}
