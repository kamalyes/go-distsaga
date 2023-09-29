/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 09:17:05
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:57:33
 * @FilePath: \go-distsaga\tcc\executor_test.go
 * @Description: TCC 执行器测试
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

func TestTCCExecutor_Execute_Success(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var tryCalled, confirmCalled, cancelCalled bool

	branches := []*Branch{
		NewBranch("branch1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				tryCalled = true
				return distsaga.StepResult{Data: map[string]string{"reserved": "100"}}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				confirmCalled = true
				return nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				cancelCalled = true
				return nil
			},
		).WithTimeout(3 * time.Second),
	}

	tx := distsaga.NewTransaction("tcc-success", distsaga.ModeTCC)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, branches)

	require.NoError(t, err)
	assert.True(t, tryCalled)
	assert.True(t, confirmCalled)
	assert.False(t, cancelCalled)
	assert.Equal(t, distsaga.StateCommitted, result.State)
}

func TestTCCExecutor_Execute_MultipleBranches(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var confirmOrder []string

	branches := []*Branch{
		NewBranch("branch1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				confirmOrder = append(confirmOrder, "confirm1")
				return nil
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
		),
		NewBranch("branch2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				confirmOrder = append(confirmOrder, "confirm2")
				return nil
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
		),
	}

	tx := distsaga.NewTransaction("tcc-multi", distsaga.ModeTCC)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, branches)

	require.NoError(t, err)
	assert.Equal(t, distsaga.StateCommitted, result.State)
	assert.Equal(t, []string{"confirm1", "confirm2"}, confirmOrder)
}

func TestTCCExecutor_Execute_TryFailure(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var cancelOrder []string

	branches := []*Branch{
		NewBranch("branch1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
			func(ctx context.Context, result distsaga.StepResult) error {
				cancelOrder = append(cancelOrder, "cancel1")
				return nil
			},
		),
		NewBranch("branch2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, errors.New("try failed")
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
			func(ctx context.Context, result distsaga.StepResult) error {
				cancelOrder = append(cancelOrder, "cancel2")
				return nil
			},
		),
	}

	tx := distsaga.NewTransaction("tcc-try-fail", distsaga.ModeTCC)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, branches)

	assert.Error(t, err)
	assert.Equal(t, distsaga.StateRolledback, result.State)
	assert.Equal(t, []string{"cancel1"}, cancelOrder)
}

func TestTCCExecutor_Execute_CancelFailure(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	branches := []*Branch{
		NewBranch("branch1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
			func(ctx context.Context, result distsaga.StepResult) error {
				return errors.New("cancel failed")
			},
		),
		NewBranch("branch2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, errors.New("try failed")
			},
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
			func(ctx context.Context, result distsaga.StepResult) error { return nil },
		),
	}

	tx := distsaga.NewTransaction("tcc-cancel-fail", distsaga.ModeTCC)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, branches)

	assert.Error(t, err)
	assert.Equal(t, distsaga.StateSuspended, result.State)
}
