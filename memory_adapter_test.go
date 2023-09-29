/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 08:55:19
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-28 13:05:07
 * @FilePath: \go-distsaga\memory_adapter_test.go
 * @Description: 内存适配器测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryAdapter_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	t.Run("保存和加载事务", func(t *testing.T) {
		tx := NewTransaction("test", ModeSaga)

		err := adapter.SaveTransaction(ctx, tx)
		require.NoError(t, err)

		loaded, err := adapter.LoadTransaction(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, tx.ID, loaded.ID)
		assert.Equal(t, tx.Name, loaded.Name)
		assert.Equal(t, tx.Mode, loaded.Mode)
		assert.Equal(t, tx.State, loaded.State)
	})

	t.Run("加载不存在的事务", func(t *testing.T) {
		_, err := adapter.LoadTransaction(ctx, "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
	})

	t.Run("重复保存事务", func(t *testing.T) {
		tx := NewTransaction("dup", ModeSaga)
		err := adapter.SaveTransaction(ctx, tx)
		require.NoError(t, err)

		err = adapter.SaveTransaction(ctx, tx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateOperation)
	})
}

func TestMemoryAdapter_UpdateTransactionState(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	t.Run("更新事务状态", func(t *testing.T) {
		tx := NewTransaction("test", ModeSaga)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))

		err := adapter.UpdateTransactionState(ctx, tx.ID, StateRunning, nil)
		require.NoError(t, err)

		loaded, err := adapter.LoadTransaction(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, StateRunning, loaded.State)
	})

	t.Run("更新不存在的事务", func(t *testing.T) {
		err := adapter.UpdateTransactionState(ctx, "nonexistent", StateRunning, nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
	})

	t.Run("更新事务状态并附加数据", func(t *testing.T) {
		tx := NewTransaction("test-payload", ModeSaga)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))

		updates := map[string]interface{}{"key": "value"}
		err := adapter.UpdateTransactionState(ctx, tx.ID, StateRunning, updates)
		require.NoError(t, err)

		loaded, err := adapter.LoadTransaction(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, StateRunning, loaded.State)
		assert.Equal(t, "value", loaded.Payload["key"])
	})
}

func TestMemoryAdapter_DeleteTransaction(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	t.Run("删除事务", func(t *testing.T) {
		tx := NewTransaction("test", ModeSaga)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))

		err := adapter.DeleteTransaction(ctx, tx.ID)
		require.NoError(t, err)

		_, err = adapter.LoadTransaction(ctx, tx.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrTransactionNotFound)
	})

	t.Run("删除不存在的事务（不报错）", func(t *testing.T) {
		err := adapter.DeleteTransaction(ctx, "nonexistent")
		assert.NoError(t, err)
	})
}

func TestMemoryAdapter_LoadByState(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	t.Run("按状态加载事务", func(t *testing.T) {
		tx1 := NewTransaction("pending-1", ModeSaga)
		tx2 := NewTransaction("pending-2", ModeSaga)
		tx3 := NewTransaction("running-1", ModeSaga)
		tx3.State = StateRunning

		require.NoError(t, adapter.SaveTransaction(ctx, tx1))
		require.NoError(t, adapter.SaveTransaction(ctx, tx2))
		require.NoError(t, adapter.SaveTransaction(ctx, tx3))

		pendingTxs, err := adapter.LoadByState(ctx, StatePending, 0)
		require.NoError(t, err)
		assert.Len(t, pendingTxs, 2)

		runningTxs, err := adapter.LoadByState(ctx, StateRunning, 0)
		require.NoError(t, err)
		assert.Len(t, runningTxs, 1)
	})

	t.Run("按状态加载事务（限制数量）", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		for i := 0; i < 5; i++ {
			tx := NewTransaction("limit-test", ModeSaga)
			require.NoError(t, adapter.SaveTransaction(ctx, tx))
		}

		txs, err := adapter.LoadByState(ctx, StatePending, 3)
		require.NoError(t, err)
		assert.Len(t, txs, 3)
	})
}

func TestMemoryAdapter_WithCtx(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	t.Run("SaveTransactionWithCtx", func(t *testing.T) {
		tx := NewTransaction("ctx-test", ModeSaga)
		err := adapter.SaveTransactionWithCtx(ctx, tx)
		require.NoError(t, err)
	})

	t.Run("LoadTransactionWithCtx", func(t *testing.T) {
		tx := NewTransaction("ctx-load", ModeSaga)
		require.NoError(t, adapter.SaveTransactionWithCtx(ctx, tx))

		loaded, err := adapter.LoadTransactionWithCtx(ctx, tx.ID)
		require.NoError(t, err)
		assert.Equal(t, tx.ID, loaded.ID)
	})

	t.Run("UpdateTransactionStateWithCtx", func(t *testing.T) {
		tx := NewTransaction("ctx-update", ModeSaga)
		require.NoError(t, adapter.SaveTransactionWithCtx(ctx, tx))

		err := adapter.UpdateTransactionStateWithCtx(ctx, tx.ID, StateRunning, nil)
		require.NoError(t, err)
	})

	t.Run("DeleteTransactionWithCtx", func(t *testing.T) {
		tx := NewTransaction("ctx-delete", ModeSaga)
		require.NoError(t, adapter.SaveTransactionWithCtx(ctx, tx))

		err := adapter.DeleteTransactionWithCtx(ctx, tx.ID)
		require.NoError(t, err)
	})
}
