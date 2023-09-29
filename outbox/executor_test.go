/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 10:33:51
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-28 17:55:19
 * @FilePath: \go-distsaga\outbox\executor_test.go
 * @Description: Outbox 执行器测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxExecutor_Execute_Success(t *testing.T) {
	t.Run("业务操作 + 消息发送成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		var bizOpCalled bool

		messages := []*Message{
			{ID: "msg-001", Target: "https://example.com/api", Body: []byte(`{"event":"test"}`)},
		}

		tx := distsaga.NewTransaction("outbox-success", distsaga.ModeOutbox)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, messages, func(ctx context.Context) error {
			bizOpCalled = true
			return nil
		})

		require.NoError(t, err)
		assert.True(t, bizOpCalled)
		assert.Equal(t, distsaga.StateCommitted, result.State)
		assert.Len(t, result.OutboxMsgs, 1)
		assert.Equal(t, distsaga.StateCommitted, result.OutboxMsgs[0].Status)
		assert.True(t, result.OutboxMsgs[0].Confirmed)
	})
}

func TestOutboxExecutor_Execute_BusinessOpFailure(t *testing.T) {
	t.Run("业务操作失败", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		messages := []*Message{
			{ID: "msg-001", Target: "https://example.com/api", Body: []byte(`{}`)},
		}

		tx := distsaga.NewTransaction("outbox-biz-fail", distsaga.ModeOutbox)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, messages, func(ctx context.Context) error {
			return errors.New("business op failed")
		})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOutboxExecutor_Execute_NoBusinessOp(t *testing.T) {
	t.Run("无业务操作直接发送消息", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		messages := []*Message{
			{ID: "msg-001", Target: "https://example.com/api", Body: []byte(`{}`)},
		}

		tx := distsaga.NewTransaction("outbox-no-biz", distsaga.ModeOutbox)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, messages, nil)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}

func TestOutboxExecutor_Execute_MultipleMessages(t *testing.T) {
	t.Run("多消息发送", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		messages := []*Message{
			{ID: "msg-001", Target: "https://svc-a/api", Body: []byte(`{}`)},
			{ID: "msg-002", Target: "https://svc-b/api", Body: []byte(`{}`)},
			{ID: "msg-003", Target: "https://svc-c/api", Body: []byte(`{}`)},
		}

		tx := distsaga.NewTransaction("outbox-multi", distsaga.ModeOutbox)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, messages, nil)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
		assert.Len(t, result.OutboxMsgs, 3)
		for _, msg := range result.OutboxMsgs {
			assert.Equal(t, distsaga.StateCommitted, msg.Status)
			assert.True(t, msg.Confirmed)
		}
	})
}

func TestOutboxExecutor_Execute_WithRetry(t *testing.T) {
	t.Run("重试策略", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		retry := distsaga.NewTestRetry()
		executor := NewExecutor(adapter, logger, retry, 5*time.Second)

		messages := []*Message{
			{ID: "msg-001", Target: "https://example.com/api", Body: []byte(`{}`)},
		}

		tx := distsaga.NewTransaction("outbox-retry", distsaga.ModeOutbox)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, messages, nil)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}

func TestMessage(t *testing.T) {
	t.Run("消息结构体", func(t *testing.T) {
		msg := &Message{
			ID:       "msg-001",
			Target:   "https://example.com/api",
			Body:     []byte(`{"event":"test"}`),
			Headers:  map[string]string{"X-Source": "order-service"},
			Delay:    5 * time.Second,
			Priority: 10,
		}
		assert.Equal(t, "msg-001", msg.ID)
		assert.Equal(t, "https://example.com/api", msg.Target)
		assert.Equal(t, "order-service", msg.Headers["X-Source"])
		assert.Equal(t, 5*time.Second, msg.Delay)
		assert.Equal(t, 10, msg.Priority)
	})
}

func TestMessageState(t *testing.T) {
	t.Run("消息状态结构体", func(t *testing.T) {
		ms := MessageState{
			ID:        "msg-001",
			Status:    distsaga.StateCommitted,
			Target:    "https://example.com/api",
			Confirmed: true,
			Retries:   0,
		}
		assert.Equal(t, "msg-001", ms.ID)
		assert.Equal(t, distsaga.StateCommitted, ms.Status)
		assert.True(t, ms.Confirmed)
	})
}

func TestWithMessage(t *testing.T) {
	t.Run("添加单条消息选项", func(t *testing.T) {
		msg := &Message{ID: "msg-001", Target: "https://example.com/api"}
		opts := NewOptions(WithMessage(msg))
		assert.Len(t, opts.Messages, 1)
	})
}

func TestWithMessages(t *testing.T) {
	t.Run("批量添加消息选项", func(t *testing.T) {
		msg1 := &Message{ID: "msg-001", Target: "https://svc-a/api"}
		msg2 := &Message{ID: "msg-002", Target: "https://svc-b/api"}
		opts := NewOptions(WithMessages(msg1, msg2))
		assert.Len(t, opts.Messages, 2)
	})
}

func TestWithBusinessOp(t *testing.T) {
	t.Run("设置业务操作选项", func(t *testing.T) {
		opts := NewOptions(WithBusinessOp(func(ctx context.Context) error {
			return nil
		}))
		assert.NotNil(t, opts.BusinessOp)
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		opts := NewOptions()
		assert.Nil(t, opts.Messages)
		assert.Nil(t, opts.BusinessOp)
	})
}
