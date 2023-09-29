/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 09:17:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-27 17:22:51
 * @FilePath: \go-distsaga\adapter_test.go
 * @Description: 适配器接口测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryAdapter_ImplementsFullStoreAdapter(t *testing.T) {
	t.Run("MemoryAdapter 实现 FullStoreAdapter", func(t *testing.T) {
		var _ FullStoreAdapter = NewMemoryAdapter()
	})
}

func TestMemoryAdapter_ImplementsTransactionalStoreAdapter(t *testing.T) {
	t.Run("接口类型断言", func(t *testing.T) {
		adapter := NewMemoryAdapter()

		var store StoreAdapter = adapter
		assert.NotNil(t, store)

		var filtered FilteredStoreAdapter = adapter
		assert.NotNil(t, filtered)

		var ctxAdapter ContextStoreAdapter = adapter
		assert.NotNil(t, ctxAdapter)

		var full FullStoreAdapter = adapter
		assert.NotNil(t, full)
	})
}

func TestMemoryAdapter_SaveTransaction_Duplicate(t *testing.T) {
	t.Run("保存重复事务返回错误", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		ctx := context.Background()

		tx := NewTransaction("dup-test", ModeSaga)
		assert.NoError(t, adapter.SaveTransaction(ctx, tx))
		assert.Error(t, adapter.SaveTransaction(ctx, tx))
	})
}

func TestMemoryAdapter_UpdateTransactionState_InvalidTransition(t *testing.T) {
	t.Run("非法状态转换返回错误", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		ctx := context.Background()

		tx := NewTransaction("invalid-trans", ModeSaga)
		assert.NoError(t, adapter.SaveTransaction(ctx, tx))

		err := adapter.UpdateTransactionState(ctx, tx.ID, StateCommitted, nil)
		assert.Error(t, err)
	})
}

func TestMemoryAdapter_UpdateTransactionStateWithCtx(t *testing.T) {
	t.Run("带上下文更新事务状态", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		ctx := context.Background()

		tx := NewTransaction("ctx-update", ModeSaga)
		assert.NoError(t, adapter.SaveTransaction(ctx, tx))

		err := adapter.UpdateTransactionStateWithCtx(ctx, tx.ID, StateRunning, map[string]interface{}{"key": "val"})
		assert.NoError(t, err)

		loaded, err := adapter.LoadTransaction(ctx, tx.ID)
		assert.NoError(t, err)
		assert.Equal(t, StateRunning, loaded.State)
		assert.Equal(t, "val", loaded.Payload["key"])
	})
}

func TestMemoryAdapter_DeleteTransactionWithCtx(t *testing.T) {
	t.Run("带上下文删除事务", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		ctx := context.Background()

		tx := NewTransaction("ctx-del", ModeSaga)
		assert.NoError(t, adapter.SaveTransaction(ctx, tx))
		assert.NoError(t, adapter.DeleteTransactionWithCtx(ctx, tx.ID))

		_, err := adapter.LoadTransaction(ctx, tx.ID)
		assert.Error(t, err)
	})
}

func TestMemoryAdapter_LoadByState_Empty(t *testing.T) {
	t.Run("按状态加载空结果", func(t *testing.T) {
		adapter := NewMemoryAdapter()
		ctx := context.Background()

		txs, err := adapter.LoadByState(ctx, StateCommitted, 0)
		assert.NoError(t, err)
		assert.Len(t, txs, 0)
	})
}
