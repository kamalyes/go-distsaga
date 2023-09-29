/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 15:51:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 19:28:51
 * @FilePath: \go-distsaga\saga\step_test.go
 * @Description: SAGA 步骤和选项测试
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

func TestNewStep(t *testing.T) {
	t.Run("创建步骤", func(t *testing.T) {
		step := NewStep("test-step",
			func(ctx Context) (distsaga.StepResult, error) {
				return distsaga.StepResult{}, nil
			},
			func(ctx Context, result distsaga.StepResult) error {
				return nil
			},
		)
		assert.Equal(t, "test-step", step.Name)
		assert.NotNil(t, step.Action)
		assert.NotNil(t, step.Compensate)
		assert.Equal(t, time.Duration(0), step.Timeout)
		assert.Equal(t, 0, step.Retries)
	})
}

func TestStep_WithTimeout(t *testing.T) {
	t.Run("设置步骤超时", func(t *testing.T) {
		step := NewStep("timeout-step", nil, nil).WithTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, step.Timeout)
	})
}

func TestStep_WithRetries(t *testing.T) {
	t.Run("设置步骤重试次数", func(t *testing.T) {
		step := NewStep("retry-step", nil, nil).WithRetries(3)
		assert.Equal(t, 3, step.Retries)
	})
}

func TestStepState(t *testing.T) {
	t.Run("步骤执行状态", func(t *testing.T) {
		ss := StepState{
			Name:   "step-1",
			Status: distsaga.StateCommitted,
			Result: distsaga.StepResult{Data: map[string]string{"key": "val"}},
		}
		assert.Equal(t, "step-1", ss.Name)
		assert.Equal(t, distsaga.StateCommitted, ss.Status)
		assert.Equal(t, "val", ss.Result.Data["key"])
	})
}

func TestWithSteps(t *testing.T) {
	t.Run("添加步骤选项", func(t *testing.T) {
		step1 := NewStep("s1", nil, nil)
		step2 := NewStep("s2", nil, nil)
		opts := NewOptions(WithSteps(step1, step2))
		assert.Len(t, opts.Steps, 2)
		assert.Equal(t, "s1", opts.Steps[0].Name)
		assert.Equal(t, "s2", opts.Steps[1].Name)
	})
}

func TestWithStepTimeout(t *testing.T) {
	t.Run("设置步骤超时选项", func(t *testing.T) {
		opts := NewOptions(WithStepTimeout(10 * time.Second))
		assert.Equal(t, 10*time.Second, opts.StepTimeout)
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("默认选项", func(t *testing.T) {
		opts := NewOptions()
		assert.Nil(t, opts.Steps)
		assert.Equal(t, time.Duration(0), opts.StepTimeout)
	})
}

func TestSagaExecutor_Execute_WithRetry(t *testing.T) {
	t.Run("重试成功", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		retry := distsaga.NewTestRetry()
		executor := NewExecutor(adapter, logger, retry, 5*time.Second)

		var attempts int
		steps := []*Step{
			NewStep("retry-step",
				func(ctx Context) (distsaga.StepResult, error) {
					attempts++
					if attempts < 2 {
						return distsaga.StepResult{}, errors.New("temporary error")
					}
					return distsaga.StepResult{}, nil
				},
				func(ctx Context, result distsaga.StepResult) error { return nil },
			).WithRetries(3),
		}

		tx := distsaga.NewTransaction("saga-retry", distsaga.ModeSaga)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		result, err := executor.Execute(ctx, tx, steps)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})
}

func TestSagaExecutor_Execute_StepTimeout(t *testing.T) {
	t.Run("步骤超时", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		steps := []*Step{
			NewStep("timeout-step",
				func(ctx Context) (distsaga.StepResult, error) {
					<-ctx.Done()
					return distsaga.StepResult{}, ctx.Err()
				},
				func(ctx Context, result distsaga.StepResult) error { return nil },
			).WithTimeout(10 * time.Millisecond),
		}

		tx := distsaga.NewTransaction("saga-timeout", distsaga.ModeSaga)
		require.NoError(t, adapter.SaveTransaction(ctx, tx))
		_, err := executor.Execute(ctx, tx, steps)
		assert.Error(t, err)
	})
}

func TestSagaExecutor_Execute_AdapterError(t *testing.T) {
	t.Run("适配器更新状态失败", func(t *testing.T) {
		ctx := context.Background()
		adapter := distsaga.NewMemoryAdapter()
		logger := distsaga.NewDefaultLogger()

		executor := NewExecutor(adapter, logger, nil, 5*time.Second)

		tx := distsaga.NewTransaction("saga-adapter-err", distsaga.ModeSaga)
		adapter.SaveTransaction(ctx, tx)
		adapter.DeleteTransaction(ctx, tx.ID)

		steps := []*Step{
			NewStep("step1",
				func(ctx Context) (distsaga.StepResult, error) {
					return distsaga.StepResult{}, nil
				},
				func(ctx Context, result distsaga.StepResult) error { return nil },
			),
		}

		_, err := executor.Execute(ctx, tx, steps)
		assert.Error(t, err)
	})
}
