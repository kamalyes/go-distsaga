/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 10:33:51
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-28 17:55:19
 * @FilePath: \go-distsaga\outbox\executor.go
 * @Description: Outbox 执行器 - 两阶段消息（Better Outbox）
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package outbox

import (
	"context"
	"fmt"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// BusinessOp 业务操作函数类型
type BusinessOp func(ctx context.Context) error

// OutboxExecutor Outbox 事务执行器
// 两阶段消息模式：先执行业务操作，再发送消息，确保最终一致性
// 比 Outbox 更优雅的解决方案，无需轮询即可保证消息不丢失
type OutboxExecutor struct {
	adapter distsaga.StoreAdapter // 存储适配器
	logger  logger.ILogger        // 日志记录器
	retry   *retry.Retry          // 重试策略
	timeout time.Duration         // 默认超时时间
}

// NewExecutor 创建 Outbox 执行器
func NewExecutor(adapter distsaga.StoreAdapter, l logger.ILogger, r *retry.Retry, timeout time.Duration) *OutboxExecutor {
	return &OutboxExecutor{
		adapter: adapter,
		logger:  l,
		retry:   r,
		timeout: timeout,
	}
}

// Execute 执行 Outbox 事务
// 1. 状态转换 PENDING → SUBMITTING
// 2. 执行业务操作（businessOp）
// 3. 业务操作成功，状态转换 SUBMITTING → SUBMITTED
// 4. 依次发送每条消息
// 5. 全部消息发送成功，状态转换 SUBMITTED → COMMITTED
func (e *OutboxExecutor) Execute(ctx context.Context, tx *distsaga.Transaction, messages []*Message, businessOp BusinessOp) (*distsaga.Transaction, error) {
	if err := tx.TransitionTo(distsaga.StateSubmitting); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_submitting", err)
	}

	// 执行业务操作
	if businessOp != nil {
		if err := businessOp(ctx); err != nil {
			e.logger.Errorf("Outbox business operation failed: %v", err)
			if failErr := tx.TransitionTo(distsaga.StateFailed); failErr != nil {
				return nil, failErr
			}
			_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
			return nil, fmt.Errorf("%w: business op: %v", distsaga.ErrPrepareFailed, err)
		}
	}

	// 业务操作成功，标记为已提交
	if err := tx.TransitionTo(distsaga.StateSubmitted); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_submitted", err)
	}

	// 记录消息状态
	for _, msg := range messages {
		msgState := MessageState{
			ID:     msg.ID,
			Status: distsaga.StatePending,
			Target: msg.Target,
		}
		tx.OutboxMsgs = append(tx.OutboxMsgs, distsaga.OutboxMessageStateItem{
			ID:        msgState.ID,
			Status:    msgState.Status,
			Target:    msgState.Target,
			SentAt:    msgState.SentAt,
			Confirmed: msgState.Confirmed,
			Retries:   msgState.Retries,
		})
	}

	// 发送消息
	if err := e.sendMessages(ctx, tx, messages); err != nil {
		return tx, err
	}

	// 全部消息发送成功，标记事务为已提交
	if err := tx.TransitionTo(distsaga.StateCommitted); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_committed", err)
	}

	e.logger.Infof("Outbox transaction [%s] committed", tx.ID)
	return tx, nil
}

// sendMessages 依次发送所有消息
// 消息发送失败时，事务进入 SUSPENDED 状态，需人工干预或恢复模块重试
func (e *OutboxExecutor) sendMessages(ctx context.Context, tx *distsaga.Transaction, messages []*Message) error {
	for i, msg := range messages {
		var err error
		if e.retry != nil {
			err = e.retry.Do(func() error {
				return e.sendMessage(ctx, msg)
			})
		} else {
			err = e.sendMessage(ctx, msg)
		}

		if err != nil {
			e.logger.Errorf("Outbox message [%s] send failed: %v", msg.ID, err)
			tx.OutboxMsgs[i].Status = distsaga.StateFailed
			tx.OutboxMsgs[i].Retries++

			if suspendErr := tx.TransitionTo(distsaga.StateSuspended); suspendErr != nil {
				return suspendErr
			}
			_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
			return fmt.Errorf("%w: message=%s: %v", distsaga.ErrCommitFailed, msg.ID, err)
		}

		tx.OutboxMsgs[i].Status = distsaga.StateCommitted
		tx.OutboxMsgs[i].SentAt = time.Now()
		tx.OutboxMsgs[i].Confirmed = true
		e.logger.Infof("Outbox message [%s] sent to %s", msg.ID, msg.Target)
	}

	return nil
}

// sendMessage 发送单条消息（占位实现，实际由适配器或消息中间件完成）
func (e *OutboxExecutor) sendMessage(_ context.Context, msg *Message) error {
	e.logger.Infof("Outbox: sending message [%s] to %s", msg.ID, msg.Target)
	return nil
}
