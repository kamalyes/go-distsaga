/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 09:28:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 01:01:15
 * @FilePath: \go-distsaga\notifier.go
 * @Description: TransactionNotifier 接口定义 - 参考 go-casbin/policy/pubsub.go 的通知机制
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import "context"

// ==================== 事务事件类型定义 ====================

// TransactionEventType 事务事件类型
type TransactionEventType string

const (
	EventTransactionCreated      TransactionEventType = "created"          // 事务已创建
	EventTransactionRunning      TransactionEventType = "running"          // 事务执行中
	EventTransactionCommitted    TransactionEventType = "committed"        // 事务已提交
	EventTransactionCompensating TransactionEventType = "compensating"     // 事务补偿中
	EventTransactionCompensated  TransactionEventType = "compensated"      // 事务已补偿
	EventTransactionRolledback   TransactionEventType = "rolledback"       // 事务已回滚
	EventTransactionFailed       TransactionEventType = "failed"           // 事务失败
	EventTransactionSuspended    TransactionEventType = "suspended"        // 事务挂起（需人工干预）
	EventStepCompleted           TransactionEventType = "step_completed"   // 步骤已完成
	EventStepFailed              TransactionEventType = "step_failed"      // 步骤失败
	EventStepCompensated         TransactionEventType = "step_compensated" // 步骤已补偿
)

// TransactionEvent 事务事件
// 用于在分布式事务各参与方之间传递事务状态变更信息
type TransactionEvent struct {
	TxID      string                 `json:"tx_id"`               // 事务 ID
	TxName    string                 `json:"tx_name"`             // 事务名称
	Mode      TxMode                 `json:"mode"`                // 事务模式（SAGA/TCC/XA/Workflow/Outbox）
	EventType TransactionEventType   `json:"event_type"`          // 事件类型
	State     TxState                `json:"state"`               // 当前事务状态
	StepName  string                 `json:"step_name,omitempty"` // 关联步骤名称（可选）
	Payload   map[string]interface{} `json:"payload,omitempty"`   // 附加数据（可选）
}

// TransactionEventHandler 事务事件处理函数
type TransactionEventHandler func(ctx context.Context, event *TransactionEvent) error

// TransactionNotifier 事务通知器接口
// 用于发布和订阅事务事件，支持跨服务通信
// Redis 适配器基于 Pub/Sub 实现，内存适配器为空实现
type TransactionNotifier interface {
	Publish(ctx context.Context, event *TransactionEvent) error           // 发布事务事件
	Subscribe(ctx context.Context, handler TransactionEventHandler) error // 订阅事务事件
	Unsubscribe() error                                                   // 取消订阅
	Close() error                                                         // 关闭通知器
}

// MemoryNotifier 内存事务通知器
// 用于测试和简单场景，事件存储在内存中
type MemoryNotifier struct {
	events   []*TransactionEvent
	handlers []TransactionEventHandler
}

// NewMemoryNotifier 创建内存通知器
func NewMemoryNotifier() *MemoryNotifier {
	return &MemoryNotifier{}
}

// Publish 发布事务事件到内存
func (n *MemoryNotifier) Publish(_ context.Context, event *TransactionEvent) error {
	n.events = append(n.events, event)
	return nil
}

// Subscribe 注册事件处理函数
func (n *MemoryNotifier) Subscribe(_ context.Context, handler TransactionEventHandler) error {
	n.handlers = append(n.handlers, handler)
	return nil
}

// Unsubscribe 取消订阅
func (n *MemoryNotifier) Unsubscribe() error { return nil }

// Close 关闭通知器
func (n *MemoryNotifier) Close() error { return nil }
