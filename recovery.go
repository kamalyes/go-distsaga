/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 15:09:22
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 10:05:52
 * @FilePath: \go-distsaga\recovery.go
 * @Description: 崩溃恢复 - 定时扫描待恢复事务并重试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"sync"
	"time"

	"github.com/kamalyes/go-logger"
)

// RecoveryHandler 恢复处理函数
// 接收待恢复的事务，执行恢复逻辑
type RecoveryHandler func(ctx context.Context, tx *Transaction) error

// Recovery 崩溃恢复器
// 定时扫描存储中处于 SUSPENDED / COMPENSATING / CONFIRMING 等状态的事务
// 并调用注册的 RecoveryHandler 进行恢复
type Recovery struct {
	adapter  StoreAdapter   // 存储适配器
	logger   logger.ILogger // 日志记录器
	interval time.Duration  // 扫描间隔
	maxAge   time.Duration  // 事务最大年龄（超过则跳过）
	handler  RecoveryHandler // 恢复处理函数
	stopCh   chan struct{}  // 停止信号通道
	once     sync.Once      // 确保 Stop 只执行一次
	running  bool           // 是否正在运行
	mu       sync.Mutex     // 保护 running 的并发安全
}

// NewRecovery 创建崩溃恢复器
func NewRecovery(adapter StoreAdapter, l logger.ILogger, interval, maxAge time.Duration) *Recovery {
	return &Recovery{
		adapter:  adapter,
		logger:   l,
		interval: interval,
		maxAge:   maxAge,
		stopCh:   make(chan struct{}),
	}
}

// RegisterHandler 注册恢复处理函数
func (r *Recovery) RegisterHandler(handler RecoveryHandler) {
	r.handler = handler
}

// Start 启动崩溃恢复
// 启动后台 goroutine 定时扫描待恢复事务
func (r *Recovery) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = true
	r.mu.Unlock()

	r.logger.Infof("Recovery started (interval=%s, maxAge=%s)", r.interval, r.maxAge)

	go r.run(ctx)

	return nil
}

// Stop 停止崩溃恢复
func (r *Recovery) Stop() {
	r.once.Do(func() {
		close(r.stopCh)
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
		r.logger.Infof("Recovery stopped")
	})
}

// run 后台扫描循环
// 定时触发 scan，支持 context 取消和 stopCh 停止
func (r *Recovery) run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Infof("Recovery context cancelled")
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			if err := r.scan(ctx); err != nil {
				r.logger.Errorf("Recovery scan failed: %v", err)
			}
		}
	}
}

// scan 扫描待恢复事务
// 遍历所有可恢复状态，从适配器加载事务并调用恢复处理函数
func (r *Recovery) scan(ctx context.Context) error {
	filtered, ok := r.adapter.(FilteredStoreAdapter)
	if !ok {
		return nil
	}

	// 可恢复的状态列表
	recoverableStates := []TxState{StateSuspended, StateCompensating, StateConfirming, StateCanceling, StateCommitting, StateAborting}

	for _, state := range recoverableStates {
		transactions, err := filtered.LoadByState(ctx, state, 100)
		if err != nil {
			r.logger.Errorf("Recovery load state %s failed: %v", state, err)
			continue
		}

		for _, tx := range transactions {
			// 跳过超龄事务
			if r.isExpired(tx) {
				r.logger.Warnf("Recovery skipping expired transaction [%s] (created=%s)", tx.ID, tx.CreatedAt.Format(time.DateTime))
				continue
			}

			if err := r.recover(ctx, tx); err != nil {
				r.logger.Errorf("Recovery failed for transaction [%s]: %v", tx.ID, err)
			}
		}
	}

	return nil
}

// recover 恢复单个事务
// 调用注册的 RecoveryHandler 执行恢复逻辑
func (r *Recovery) recover(ctx context.Context, tx *Transaction) error {
	r.logger.Infof("Recovering transaction [%s] in state %s (mode=%s)", tx.ID, tx.State, tx.Mode)

	if r.handler != nil {
		return r.handler(ctx, tx)
	}

	r.logger.Warnf("No recovery handler registered, skipping transaction [%s]", tx.ID)
	return nil
}

// isExpired 判断事务是否超龄
// 超龄事务不再尝试恢复，需人工干预
func (r *Recovery) isExpired(tx *Transaction) bool {
	if r.maxAge <= 0 {
		return false
	}
	return time.Since(tx.CreatedAt) > r.maxAge
}
