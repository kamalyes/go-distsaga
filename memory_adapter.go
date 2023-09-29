/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 15:31:02
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-28 11:19:22
 * @FilePath: \go-distsaga\memory_adapter.go
 * @Description: 内置内存适配器 - 用于测试和简单场景，实现 FullStoreAdapter 接口
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"fmt"
	"sync"
)

// 编译时检查 MemoryAdapter 实现了 FullStoreAdapter 接口
var _ FullStoreAdapter = (*MemoryAdapter)(nil)

// MemoryAdapter 内存事务存储适配器
// 使用 sync.RWMutex 保护并发安全，数据存储在内存 map 中
// 适用于测试和简单场景，不支持持久化
type MemoryAdapter struct {
	mu   sync.RWMutex            // 读写锁，保护 data 的并发安全
	data map[string]*Transaction // 事务存储，key 为事务 ID
}

// NewMemoryAdapter 创建内存适配器
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{
		data: make(map[string]*Transaction),
	}
}

// SaveTransaction 保存事务到内存
// 事务 ID 重复时返回 ErrDuplicateOperation
func (a *MemoryAdapter) SaveTransaction(_ context.Context, tx *Transaction) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.data[tx.ID]; exists {
		return fmt.Errorf("%w: transaction %s already exists", ErrDuplicateOperation, tx.ID)
	}
	cp := *tx
	a.data[tx.ID] = &cp
	return nil
}

// LoadTransaction 从内存加载事务
// 事务不存在时返回 ErrTransactionNotFound
func (a *MemoryAdapter) LoadTransaction(_ context.Context, txID string) (*Transaction, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	tx, ok := a.data[txID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}
	cp := *tx
	return &cp, nil
}

// UpdateTransactionState 更新事务状态
// 同时更新 Payload 中的附加数据
func (a *MemoryAdapter) UpdateTransactionState(_ context.Context, txID string, state TxState, updates map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, ok := a.data[txID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}
	if err := tx.TransitionTo(state); err != nil {
		return err
	}
	if tx.Payload == nil {
		tx.Payload = make(map[string]interface{})
	}
	for k, v := range updates {
		tx.Payload[k] = v
	}
	return nil
}

// DeleteTransaction 从内存删除事务
func (a *MemoryAdapter) DeleteTransaction(_ context.Context, txID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.data, txID)
	return nil
}

// SaveTransactionWithCtx 带上下文保存事务（委托给 SaveTransaction）
func (a *MemoryAdapter) SaveTransactionWithCtx(ctx context.Context, tx *Transaction) error {
	return a.SaveTransaction(ctx, tx)
}

// LoadTransactionWithCtx 带上下文加载事务（委托给 LoadTransaction）
func (a *MemoryAdapter) LoadTransactionWithCtx(ctx context.Context, txID string) (*Transaction, error) {
	return a.LoadTransaction(ctx, txID)
}

// UpdateTransactionStateWithCtx 带上下文更新事务状态（委托给 UpdateTransactionState）
func (a *MemoryAdapter) UpdateTransactionStateWithCtx(ctx context.Context, txID string, state TxState, updates map[string]interface{}) error {
	return a.UpdateTransactionState(ctx, txID, state, updates)
}

// DeleteTransactionWithCtx 带上下文删除事务（委托给 DeleteTransaction）
func (a *MemoryAdapter) DeleteTransactionWithCtx(ctx context.Context, txID string) error {
	return a.DeleteTransaction(ctx, txID)
}

// LoadByState 按状态加载事务列表
// limit > 0 时限制返回数量
func (a *MemoryAdapter) LoadByState(_ context.Context, state TxState, limit int) ([]*Transaction, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var result []*Transaction
	for _, tx := range a.data {
		if tx.State == state {
			cp := *tx
			result = append(result, &cp)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}
