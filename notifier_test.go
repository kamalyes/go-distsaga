/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 09:28:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 01:02:09
 * @FilePath: \go-distsaga\notifier_test.go
 * @Description: 事务通知器接口测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionEvent(t *testing.T) {
	t.Run("创建事务事件", func(t *testing.T) {
		event := &TransactionEvent{
			TxID:      "tx-001",
			TxName:    "order-create",
			Mode:      ModeSaga,
			EventType: EventTransactionCommitted,
			State:     StateCommitted,
			StepName:  "step1",
			Payload:   map[string]interface{}{"key": "value"},
		}
		assert.Equal(t, "tx-001", event.TxID)
		assert.Equal(t, "order-create", event.TxName)
		assert.Equal(t, ModeSaga, event.Mode)
		assert.Equal(t, EventTransactionCommitted, event.EventType)
		assert.Equal(t, StateCommitted, event.State)
		assert.Equal(t, "step1", event.StepName)
		assert.Equal(t, "value", event.Payload["key"])
	})
}

func TestTransactionNotifier_Interface(t *testing.T) {
	t.Run("MemoryNotifier 实现 TransactionNotifier 接口", func(t *testing.T) {
		var _ TransactionNotifier = &MemoryNotifier{}
	})
}

func TestMemoryNotifier(t *testing.T) {
	t.Run("发布和订阅事件", func(t *testing.T) {
		notifier := &MemoryNotifier{}
		ctx := context.Background()

		var receivedEvent *TransactionEvent
		notifier.Subscribe(ctx, func(_ context.Context, event *TransactionEvent) error {
			receivedEvent = event
			return nil
		})

		event := &TransactionEvent{
			TxID:      "tx-001",
			EventType: EventTransactionCommitted,
			State:     StateCommitted,
		}
		err := notifier.Publish(ctx, event)
		assert.NoError(t, err)
		assert.Len(t, notifier.events, 1)

		err = notifier.Unsubscribe()
		assert.NoError(t, err)

		err = notifier.Close()
		assert.NoError(t, err)

		_ = receivedEvent
	})
}

func TestTransactionEventTypes(t *testing.T) {
	t.Run("所有事件类型定义", func(t *testing.T) {
		assert.Equal(t, TransactionEventType("created"), EventTransactionCreated)
		assert.Equal(t, TransactionEventType("running"), EventTransactionRunning)
		assert.Equal(t, TransactionEventType("committed"), EventTransactionCommitted)
		assert.Equal(t, TransactionEventType("compensating"), EventTransactionCompensating)
		assert.Equal(t, TransactionEventType("compensated"), EventTransactionCompensated)
		assert.Equal(t, TransactionEventType("rolledback"), EventTransactionRolledback)
		assert.Equal(t, TransactionEventType("failed"), EventTransactionFailed)
		assert.Equal(t, TransactionEventType("suspended"), EventTransactionSuspended)
		assert.Equal(t, TransactionEventType("step_completed"), EventStepCompleted)
		assert.Equal(t, TransactionEventType("step_failed"), EventStepFailed)
		assert.Equal(t, TransactionEventType("step_compensated"), EventStepCompensated)
	})
}
