/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 13:09:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 00:58:21
 * @FilePath: \go-distsaga\tcc\branch_test.go
 * @Description: TCC 分支和选项测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package tcc

import (
	"context"
	"errors"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBranch(t *testing.T) {
	t.Run("创建分支", func(t *testing.T) {
		branch := NewBranch("test-branch",
			func(ctx Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx Context, result distsaga.StepResult) error { return nil },
			func(ctx Context, result distsaga.StepResult) error { return nil },
		)
		assert.Equal(t, "test-branch", branch.Name)
		assert.NotNil(t, branch.Try)
		assert.NotNil(t, branch.Confirm)
		assert.NotNil(t, branch.Cancel)
		assert.Equal(t, time.Duration(0), branch.Timeout)
		assert.Equal(t, 0, branch.Retries)
	})
}

func TestBranch_WithTimeout(t *testing.T) {
	t.Run("设置分支超时", func(t *testing.T) {
		branch := NewBranch("b", nil, nil, nil).WithTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, branch.Timeout)
	})
}

func TestBranch_WithRetries(t *testing.T) {
	t.Run("设置分支重试次数", func(t *testing.T) {
		branch := NewBranch("b", nil, nil, nil).WithRetries(3)
		assert.Equal(t, 3, branch.Retries)
	})
}

func TestBranchState(t *testing.T) {
	t.Run("分支执行状态", func(t *testing.T) {
		bs := BranchState{
			Name:      "branch-1",
			Status:    distsaga.StatePrepared,
			TryResult: distsaga.StepResult{Data: map[string]string{"frozen": "100"}},
		}
		assert.Equal(t, "branch-1", bs.Name)
		assert.Equal(t, distsaga.StatePrepared, bs.Status)
		assert.Equal(t, "100", bs.TryResult.Data["frozen"])
	})
}

func TestWithBranches(t *testing.T) {
	t.Run("添加分支选项", func(t *testing.T) {
		b1 := NewBranch("b1", nil, nil, nil)
		b2 := NewBranch("b2", nil, nil, nil)
		opts := NewOptions(WithBranches(b1, b2))
		assert.Len(t, opts.Branches, 2)
		assert.Equal(t, "b1", opts.Branches[0].Name)
		assert.Equal(t, "b2", opts.Branches[1].Name)
	})
}

func TestWithStepTimeout(t *testing.T) {
	t.Run("设置分支超时选项", func(t *testing.T) {
		opts := NewOptions(WithStepTimeout(10 * time.Second))
		assert.Equal(t, 10*time.Second, opts.StepTimeout)
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		opts := NewOptions()
		assert.Nil(t, opts.Branches)
		assert.Equal(t, time.Duration(0), opts.StepTimeout)
	})
}

func TestTCCExecutor_Execute_ConfirmFailure(t *testing.T) {
	t.Run("Confirm 失败导致挂起", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		branches := []*Branch{
			NewBranch("branch1",
				func(ctx Context) (distsaga.StepResult, error) {
					return distsaga.StepResult{}, nil
				},
				func(ctx Context, result distsaga.StepResult) error {
					return errors.New("confirm failed")
				},
				func(ctx Context, result distsaga.StepResult) error { return nil },
			),
		}

		tx := distsaga.NewTransaction("tcc-confirm-fail", distsaga.ModeTCC)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		assert.Error(t, err)
		assert.Equal(t, distsaga.StateSuspended, result.State)
	})
}

func TestTCCExecutor_Execute_WithRetry(t *testing.T) {
	t.Run("Try 阶段重试成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		retry := distsaga.NewTestRetry()
		executor := NewExecutor(adapter, logger, retry, 5*time.Second)

		var tryAttempts int
		branches := []*Branch{
			NewBranch("branch1",
				func(ctx Context) (distsaga.StepResult, error) {
					tryAttempts++
					if tryAttempts < 2 {
						return distsaga.StepResult{}, errors.New("temporary error")
					}
					return distsaga.StepResult{}, nil
				},
				func(ctx Context, result distsaga.StepResult) error { return nil },
				func(ctx Context, result distsaga.StepResult) error { return nil },
			).WithRetries(3),
		}

		tx := distsaga.NewTransaction("tcc-retry", distsaga.ModeTCC)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, branches)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}
