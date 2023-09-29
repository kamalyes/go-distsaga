/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 09:17:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-27 17:22:51
 * @FilePath: \go-distsaga\adapter.go
 * @Description: StoreAdapter 接口定义 - 参考 go-casbin/policy/adapter.go 的分层接口设计
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import "context"

// ==================== 适配器接口定义 ====================

// StoreAdapter 基础事务存储适配器接口
// 所有适配器必须实现这四个核心方法：保存、加载、更新状态、删除
type StoreAdapter interface {
	SaveTransaction(ctx context.Context, tx *Transaction) error                                                   // 保存事务到存储
	LoadTransaction(ctx context.Context, txID string) (*Transaction, error)                                       // 从存储加载事务
	UpdateTransactionState(ctx context.Context, txID string, state TxState, updates map[string]interface{}) error // 更新事务状态
	DeleteTransaction(ctx context.Context, txID string) error                                                     // 删除事务
}

// FilteredStoreAdapter 支持按状态过滤加载的适配器接口
// 扩展 StoreAdapter，增加按事务状态批量加载的能力
// 适用于恢复模块按 SUSPENDED / COMPENSATING 等状态扫描待恢复事务
type FilteredStoreAdapter interface {
	StoreAdapter
	LoadByState(ctx context.Context, state TxState, limit int) ([]*Transaction, error) // 按状态加载事务列表
}

// ContextStoreAdapter 支持 Context 的事务存储适配器接口
// 扩展 StoreAdapter，增加带 context 的 CRUD 方法
// 适用于需要超时控制、链路追踪或请求取消的场景
type ContextStoreAdapter interface {
	StoreAdapter
	SaveTransactionWithCtx(ctx context.Context, tx *Transaction) error                                                   // 带上下文保存事务
	LoadTransactionWithCtx(ctx context.Context, txID string) (*Transaction, error)                                       // 带上下文加载事务
	UpdateTransactionStateWithCtx(ctx context.Context, txID string, state TxState, updates map[string]interface{}) error // 带上下文更新事务状态
	DeleteTransactionWithCtx(ctx context.Context, txID string) error                                                     // 带上下文删除事务
}

// FullStoreAdapter 全功能事务存储适配器接口
// 组合了 ContextStoreAdapter 和 FilteredStoreAdapter
// Redis 适配器和 GORM 适配器均实现了此接口
type FullStoreAdapter interface {
	ContextStoreAdapter
	FilteredStoreAdapter
}

// TransactionalStoreAdapter 支持事务的存储适配器接口
// 扩展 StoreAdapter，增加事务执行能力
// 适配器实现此接口后，可在同一个数据库事务中执行多个操作
type TransactionalStoreAdapter interface {
	StoreAdapter
	ExecuteInTransaction(ctx context.Context, fn func(StoreAdapter) error) error // 在事务中执行函数
}
