/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 08:13:29
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 20:17:05
 * @FilePath: \go-distsaga\saga\executor_test.go
 * @Description: SAGA 执行器测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSagaExecutor_Execute_Success(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var actionCalled, compensateCalled bool

	steps := []*Step{
		NewStep("step1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				actionCalled = true
				return distsaga.StepResult{Data: map[string]string{"key": "value1"}}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				compensateCalled = true
				return nil
			},
		),
	}

	tx := distsaga.NewTransaction("saga-success", distsaga.ModeSaga)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, steps)

	require.NoError(t, err)
	assert.True(t, actionCalled)
	assert.False(t, compensateCalled)
	assert.Equal(t, distsaga.StateCommitted, result.State)
	assert.Len(t, result.Steps, 1)
	assert.Equal(t, distsaga.StateCommitted, result.Steps[0].Status)
}

func TestSagaExecutor_Execute_MultipleSteps(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var executionOrder []string

	steps := []*Step{
		NewStep("step1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				executionOrder = append(executionOrder, "action1")
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				executionOrder = append(executionOrder, "compensate1")
				return nil
			},
		),
		NewStep("step2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				executionOrder = append(executionOrder, "action2")
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				executionOrder = append(executionOrder, "compensate2")
				return nil
			},
		),
		NewStep("step3",
			func(ctx context.Context) (distsaga.StepResult, error) {
				executionOrder = append(executionOrder, "action3")
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				executionOrder = append(executionOrder, "compensate3")
				return nil
			},
		),
	}

	tx := distsaga.NewTransaction("saga-multi", distsaga.ModeSaga)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, steps)

	require.NoError(t, err)
	assert.Equal(t, distsaga.StateCommitted, result.State)
	assert.Equal(t, []string{"action1", "action2", "action3"}, executionOrder)
}

func TestSagaExecutor_Execute_FailureWithCompensation(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	var compensationOrder []string

	steps := []*Step{
		NewStep("step1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				compensationOrder = append(compensationOrder, "compensate1")
				return nil
			},
		),
		NewStep("step2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				compensationOrder = append(compensationOrder, "compensate2")
				return nil
			},
		),
		NewStep("step3",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, errors.New("step3 failed")
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				compensationOrder = append(compensationOrder, "compensate3")
				return nil
			},
		),
	}

	tx := distsaga.NewTransaction("saga-fail", distsaga.ModeSaga)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, steps)

	assert.Error(t, err)
	assert.Equal(t, distsaga.StateCompensated, result.State)
	assert.Equal(t, []string{"compensate2", "compensate1"}, compensationOrder)
}

func TestSagaExecutor_Execute_CompensationFailure(t *testing.T) {
	ctx := context.Background()
	adapter := distsaga.NewMemoryAdapter()
	logger := distsaga.NewDefaultLogger()

	executor := NewExecutor(adapter, logger, nil, 5*time.Second)

	steps := []*Step{
		NewStep("step1",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				return errors.New("compensation failed")
			},
		),
		NewStep("step2",
			func(ctx context.Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, errors.New("action failed")
			},
			func(ctx context.Context, result distsaga.StepResult) error {
				return nil
			},
		),
	}

	tx := distsaga.NewTransaction("saga-comp-fail", distsaga.ModeSaga)
	require.NoError(t, adapter.SaveTransaction(ctx, tx))
	result, err := executor.Execute(ctx, tx, steps)

	assert.Error(t, err)
	assert.Equal(t, distsaga.StateSuspended, result.State)
}
