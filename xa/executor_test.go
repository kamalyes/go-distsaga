/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 18:19:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:59
 * @FilePath: \go-distsaga\xa\executor_test.go
 * @Description: XA 执行器测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package xa

import (
	"context"
	"errors"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResource struct {
	prepareErr error
	commitErr  error
	rollbackFn func(ctx context.Context) error
}

func (r *mockResource) Prepare(ctx context.Context) error { return r.prepareErr }
func (r *mockResource) Commit(ctx context.Context) error  { return r.commitErr }
func (r *mockResource) Rollback(ctx context.Context) error {
	if r.rollbackFn != nil {
		return r.rollbackFn(ctx)
	}
	return nil
}

func TestXAExecutor_Execute_Success(t *testing.T) {
	t.Run("全部 Prepare + Commit 成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		var commitOrder []string
		branches := []*Branch{
			NewBranch("branch1", &mockResource{commitErr: nil}).WithTimeout(3 * time.Second),
			NewBranch("branch2", &mockResource{
				commitErr: nil,
			}),
		}

		_ = commitOrder

		tx := distsaga.NewTransaction("xa-success", distsaga.ModeXA)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
		assert.Len(t, result.XABranches, 2)
		assert.Equal(t, distsaga.StateCommitted, result.XABranches[0].Status)
		assert.Equal(t, distsaga.StateCommitted, result.XABranches[1].Status)
	})
}

func TestXAExecutor_Execute_PrepareFailure(t *testing.T) {
	t.Run("Prepare 失败触发 Rollback", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		branches := []*Branch{
			NewBranch("branch1", &mockResource{}),
			NewBranch("branch2", &mockResource{prepareErr: errors.New("prepare failed")}),
		}

		tx := distsaga.NewTransaction("xa-prepare-fail", distsaga.ModeXA)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateRolledback, result.State)
	})
}

func TestXAExecutor_Execute_CommitFailure(t *testing.T) {
	t.Run("Commit 失败导致挂起", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		branches := []*Branch{
			NewBranch("branch1", &mockResource{commitErr: errors.New("commit failed")}),
		}

		tx := distsaga.NewTransaction("xa-commit-fail", distsaga.ModeXA)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateSuspended, result.State)
	})
}

func TestXAExecutor_Execute_RollbackFailure(t *testing.T) {
	t.Run("Rollback 失败导致挂起", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		branches := []*Branch{
			NewBranch("branch1", &mockResource{
				rollbackFn: func(ctx context.Context) error {
					return errors.New("rollback failed")
				},
			}),
			NewBranch("branch2", &mockResource{prepareErr: errors.New("prepare failed")}),
		}

		tx := distsaga.NewTransaction("xa-rollback-fail", distsaga.ModeXA)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateSuspended, result.State)
	})
}

func TestXAExecutor_Execute_WithRetry(t *testing.T) {
	t.Run("重试成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		retry := distsaga.NewTestRetry()
		executor := NewExecutor(adapter, logger, retry, 5*time.Second)

		branches := []*Branch{
			NewBranch("branch1", &mockResource{}),
		}

		tx := distsaga.NewTransaction("xa-retry", distsaga.ModeXA)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}

func TestNewBranch(t *testing.T) {
	t.Run("创建 XA 分支", func(t *testing.T) {
		branch := NewBranch("test-branch", &mockResource{})
		assert.Equal(t, "test-branch", branch.Name)
		assert.NotNil(t, branch.Resource)
	})
}

func TestBranch_WithTimeout(t *testing.T) {
	t.Run("设置分支超时", func(t *testing.T) {
		branch := NewBranch("b", &mockResource{}).WithTimeout(10 * time.Second)
		assert.Equal(t, 10*time.Second, branch.Timeout)
	})
}

func TestWithBranches(t *testing.T) {
	t.Run("添加分支选项", func(t *testing.T) {
		b1 := NewBranch("b1", &mockResource{})
		b2 := NewBranch("b2", &mockResource{})
		opts := NewOptions(WithBranches(b1, b2))
		assert.Len(t, opts.Branches, 2)
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		opts := NewOptions()
		assert.Nil(t, opts.Branches)
	})
}

func TestBranchState(t *testing.T) {
	t.Run("分支执行状态", func(t *testing.T) {
		bs := BranchState{
			Name:   "branch-1",
			Status: distsaga.StatePrepared,
		}
		assert.Equal(t, "branch-1", bs.Name)
		assert.Equal(t, distsaga.StatePrepared, bs.Status)
	})
}

func TestResource(t *testing.T) {
	t.Run("Resource 接口实现", func(t *testing.T) {
		var _ Resource = &mockResource{}
	})
}
