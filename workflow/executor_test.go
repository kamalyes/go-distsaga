/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 15:22:09
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:33
 * @FilePath: \go-distsaga\workflow\executor_test.go
 * @Description: Workflow 执行器测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowExecutor_Execute_Success(t *testing.T) {
	t.Run("Handler 成功 + OnConfirm 成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		var confirmCalled bool

		handler := func(wf *Workflow, data []byte) error {
			wf.OnConfirm(func(ctx context.Context, data []byte) error {
				confirmCalled = true
				return nil
			})
			return nil
		}

		tx := distsaga.NewTransaction("wf-success", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, handler, nil)

		require.NoError(t, err)
		assert.True(t, confirmCalled)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}

func TestWorkflowExecutor_Execute_HandlerFailure(t *testing.T) {
	t.Run("Handler 失败触发回滚", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		var rollbackOrder []string

		handler := func(wf *Workflow, data []byte) error {
			wf.OnRollback(func(ctx context.Context, data []byte) error {
				rollbackOrder = append(rollbackOrder, "rollback1")
				return nil
			})
			wf.OnRollback(func(ctx context.Context, data []byte) error {
				rollbackOrder = append(rollbackOrder, "rollback2")
				return nil
			})
			return errors.New("handler failed")
		}

		tx := distsaga.NewTransaction("wf-fail", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, handler, nil)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateCompensated, result.State)
		assert.Equal(t, []string{"rollback2", "rollback1"}, rollbackOrder)
	})
}

func TestWorkflowExecutor_Execute_RollbackFailure(t *testing.T) {
	t.Run("回滚失败导致挂起", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		handler := func(wf *Workflow, data []byte) error {
			wf.OnRollback(func(ctx context.Context, data []byte) error {
				return errors.New("rollback failed")
			})
			return errors.New("handler failed")
		}

		tx := distsaga.NewTransaction("wf-rollback-fail", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, handler, nil)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateSuspended, result.State)
	})
}

func TestWorkflowExecutor_Execute_ConfirmFailure(t *testing.T) {
	t.Run("OnConfirm 失败导致挂起", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		handler := func(wf *Workflow, data []byte) error {
			wf.OnConfirm(func(ctx context.Context, data []byte) error {
				return errors.New("confirm failed")
			})
			return nil
		}

		tx := distsaga.NewTransaction("wf-confirm-fail", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, handler, nil)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateSuspended, result.State)
	})
}

func TestWorkflowExecutor_Execute_OnCancel(t *testing.T) {
	t.Run("OnCancel 作为 OnRollback 的备选", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		var cancelCalled bool

		handler := func(wf *Workflow, data []byte) error {
			wf.OnCancel(func(ctx context.Context, data []byte) error {
				cancelCalled = true
				return nil
			})
			return errors.New("handler failed")
		}

		tx := distsaga.NewTransaction("wf-cancel", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, handler, nil)

		assert.Error(t, err)
		assert.True(t, cancelCalled)
		assert.Equal(t, distsaga.StateCompensated, result.State)
	})
}

func TestWorkflowExecutor_Execute_WithInputData(t *testing.T) {
	t.Run("传递输入数据", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, 5*time.Second)

		var receivedData []byte

		handler := func(wf *Workflow, data []byte) error {
			receivedData = data
			return nil
		}

		tx := distsaga.NewTransaction("wf-data", distsaga.ModeWorkflow)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		_, err := executor.Execute(ctx, tx, handler, []byte(`{"key":"value"}`))

		require.NoError(t, err)
		assert.Equal(t, `{"key":"value"}`, string(receivedData))
	})
}

func TestWorkflow_NewBranch(t *testing.T) {
	t.Run("创建空分支", func(t *testing.T) {
		wf := &Workflow{ID: "test"}
		branch := wf.NewBranch()
		assert.NotNil(t, branch)
	})
}

func TestWorkflow_GetBranches(t *testing.T) {
	t.Run("获取分支列表", func(t *testing.T) {
		wf := &Workflow{ID: "test"}
		wf.OnRollback(func(ctx context.Context, data []byte) error { return nil })
		wf.OnConfirm(func(ctx context.Context, data []byte) error { return nil })

		branches := wf.GetBranches()
		assert.Len(t, branches, 2)
	})
}

func TestWithHandler(t *testing.T) {
	t.Run("设置 Handler 选项", func(t *testing.T) {
		handler := func(wf *Workflow, data []byte) error { return nil }
		opts := NewOptions(WithHandler(handler))
		assert.NotNil(t, opts.Handler)
	})
}

func TestWithData(t *testing.T) {
	t.Run("设置数据选项", func(t *testing.T) {
		opts := NewOptions(WithData([]byte("test")))
		assert.Equal(t, []byte("test"), opts.Data)
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		opts := NewOptions()
		assert.Nil(t, opts.Handler)
		assert.Nil(t, opts.Data)
	})
}
