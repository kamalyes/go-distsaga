/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 15:09:22
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 10:05:52
 * @FilePath: \go-distsaga\recovery_test.go
 * @Description: 崩溃恢复测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecovery(t *testing.T) {
	t.Run("创建恢复器", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 10*time.Second, 24*time.Hour)
		assert.NotNil(t, r)
	})
}

func TestRecovery_RegisterHandler(t *testing.T) {
	t.Run("注册恢复处理函数", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 10*time.Second, 24*time.Hour)
		r.RegisterHandler(func(ctx context.Context, tx *Transaction) error {
			return nil
		})
	})
}

func TestRecovery_StartStop(t *testing.T) {
	t.Run("启动和停止恢复", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := r.Start(ctx)
		require.NoError(t, err)

		r.Stop()
	})

	t.Run("重复启动不报错", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx := context.Background()
		require.NoError(t, r.Start(ctx))
		require.NoError(t, r.Start(ctx))
		r.Stop()
	})

	t.Run("重复停止不 panic", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx := context.Background()
		require.NoError(t, r.Start(ctx))
		r.Stop()
		r.Stop()
	})
}

func TestRecovery_ScanAndRecover(t *testing.T) {
	t.Run("扫描到可恢复事务并调用 handler", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()

		tx := NewTransaction("recoverable", ModeSaga)
		tx.State = StateSuspended
		require.NoError(t, adapter.SaveTransaction(context.Background(), tx))

		var recoveredTxID string
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)
		r.RegisterHandler(func(ctx context.Context, tx *Transaction) error {
			recoveredTxID = tx.ID
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, r.Start(ctx))
		time.Sleep(300 * time.Millisecond)
		r.Stop()

		assert.Equal(t, tx.ID, recoveredTxID)
	})

	t.Run("超龄事务跳过恢复", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()

		tx := NewTransaction("expired", ModeSaga)
		tx.State = StateSuspended
		tx.CreatedAt = time.Now().Add(-48 * time.Hour)
		require.NoError(t, adapter.SaveTransaction(context.Background(), tx))

		var handlerCalled bool
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 1*time.Hour)
		r.RegisterHandler(func(ctx context.Context, tx *Transaction) error {
			handlerCalled = true
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, r.Start(ctx))
		time.Sleep(300 * time.Millisecond)
		r.Stop()

		assert.False(t, handlerCalled)
	})

	t.Run("无 handler 时不恢复", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()

		tx := NewTransaction("no-handler", ModeSaga)
		tx.State = StateSuspended
		require.NoError(t, adapter.SaveTransaction(context.Background(), tx))

		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, r.Start(ctx))
		time.Sleep(300 * time.Millisecond)
		r.Stop()
	})

	t.Run("context 取消时停止恢复", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()
		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		require.NoError(t, r.Start(ctx))
		cancel()
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("适配器不支持 FilteredStoreAdapter 时跳过扫描", func(t *testing.T) {
		logger := NewDefaultLogger()
		basicAdapter := &basicOnlyAdapter{}
		r := NewRecovery(basicAdapter, logger, 100*time.Millisecond, 24*time.Hour)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, r.Start(ctx))
		time.Sleep(200 * time.Millisecond)
		r.Stop()
	})

	t.Run("handler 返回错误时记录日志", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		logger := NewDefaultLogger()

		tx := NewTransaction("handler-err", ModeSaga)
		tx.State = StateSuspended
		require.NoError(t, adapter.SaveTransaction(context.Background(), tx))

		r := NewRecovery(adapter, logger, 100*time.Millisecond, 24*time.Hour)
		r.RegisterHandler(func(ctx context.Context, tx *Transaction) error {
			return ErrRecoveryFailed
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, r.Start(ctx))
		time.Sleep(300 * time.Millisecond)
		r.Stop()
	})
}

type basicOnlyAdapter struct{}

func (a *basicOnlyAdapter) SaveTransaction(_ context.Context, _ *Transaction) error { return nil }
func (a *basicOnlyAdapter) LoadTransaction(_ context.Context, _ string) (*Transaction, error) {
	return nil, ErrTransactionNotFound
}
func (a *basicOnlyAdapter) UpdateTransactionState(_ context.Context, _ string, _ TxState, _ map[string]interface{}) error {
	return nil
}
func (a *basicOnlyAdapter) DeleteTransaction(_ context.Context, _ string) error { return nil }
