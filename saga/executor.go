/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 11:19:02
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 17:35:08
 * @FilePath: \go-distsaga\saga\executor.go
 * @Description: SAGA 执行器 - 正向执行 + 逆序补偿
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package saga

import (
	"context"
	"fmt"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// SagaExecutor SAGA 事务执行器
// 按顺序执行正向步骤，任一步骤失败则逆序执行已完成步骤的补偿操作
type SagaExecutor struct {
	adapter distsaga.StoreAdapter // 存储适配器
	logger  logger.ILogger        // 日志记录器
	retry   *retry.Retry          // 重试策略
	timeout time.Duration         // 默认超时时间
}

// NewExecutor 创建 SAGA 执行器
func NewExecutor(adapter distsaga.StoreAdapter, l logger.ILogger, r *retry.Retry, timeout time.Duration) *SagaExecutor {
	return &SagaExecutor{
		adapter: adapter,
		logger:  l,
		retry:   r,
		timeout: timeout,
	}
}

// Execute 执行 SAGA 事务
func (e *SagaExecutor) Execute(ctx context.Context, tx *distsaga.Transaction, steps []*Step) (*distsaga.Transaction, error) {
	if err := tx.TransitionTo(distsaga.StateRunning); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_running", err)
	}

	var completedSteps []int
	for i, step := range steps {
		stepCtx, cancel := e.stepContext(ctx, step.Timeout)

		var result distsaga.StepResult
		var err error

		if e.retry != nil && step.Retries > 0 {
			r := retry.NewRetryWithCtx(stepCtx).
				SetAttemptCount(step.Retries + 1).
				SetInterval(100 * time.Millisecond)
			err = r.Do(func() error {
				var actionErr error
				result, actionErr = step.Action(stepCtx)
				return actionErr
			})
		} else {
			result, err = step.Action(stepCtx)
		}
		cancel()

		ss := StepState{
			Name:      step.Name,
			StartedAt: time.Now(),
		}

		if err != nil {
			e.logger.Errorf("SAGA step [%s] action failed: %v", step.Name, err)
			ss.Status = distsaga.StateFailed
			ss.Result = distsaga.StepResult{Error: err}
			tx.Steps = append(tx.Steps, distsaga.StepStateItem{
				Name:      ss.Name,
				Status:    ss.Status,
				Result:    ss.Result,
				StartedAt: ss.StartedAt,
				EndedAt:   ss.EndedAt,
				Retries:   ss.Retries,
			})

			if compErr := e.compensate(ctx, tx, steps, completedSteps); compErr != nil {
				return tx, fmt.Errorf("%w: action=%s compensate=%v", distsaga.ErrCompensationFailed, step.Name, compErr)
			}
			return tx, distsaga.NewTransactionError(tx.ID, step.Name, distsaga.ModeSaga, tx.State, err, "action failed, compensated")
		}

		ss.Status = distsaga.StateCommitted
		ss.Result = result
		ss.EndedAt = time.Now()
		tx.Steps = append(tx.Steps, distsaga.StepStateItem{
			Name:      ss.Name,
			Status:    ss.Status,
			Result:    ss.Result,
			StartedAt: ss.StartedAt,
			EndedAt:   ss.EndedAt,
			Retries:   ss.Retries,
		})
		completedSteps = append(completedSteps, i)

		e.logger.Infof("SAGA step [%s] completed", step.Name)
	}

	if err := tx.TransitionTo(distsaga.StateCommitted); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_committed", err)
	}

	e.logger.Infof("SAGA transaction [%s] committed", tx.ID)
	return tx, nil
}

// compensate 逆序执行补偿操作
func (e *SagaExecutor) compensate(ctx context.Context, tx *distsaga.Transaction, steps []*Step, completedIndices []int) error {
	if err := tx.TransitionTo(distsaga.StateCompensating); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_compensating", err)
	}

	var lastErr error
	for i := len(completedIndices) - 1; i >= 0; i-- {
		idx := completedIndices[i]
		step := steps[idx]
		stepResult := tx.Steps[idx].Result

		var compErr error
		if e.retry != nil {
			compErr = e.retry.Do(func() error {
				return step.Compensate(ctx, stepResult)
			})
		} else {
			compErr = step.Compensate(ctx, stepResult)
		}

		if compErr != nil {
			e.logger.Errorf("SAGA step [%s] compensate failed: %v", step.Name, compErr)
			tx.Steps[idx].Status = distsaga.StateFailed
			lastErr = compErr
		} else {
			tx.Steps[idx].Status = distsaga.StateCompensated
			e.logger.Infof("SAGA step [%s] compensated", step.Name)
		}
	}

	if lastErr != nil {
		if err := tx.TransitionTo(distsaga.StateSuspended); err != nil {
			return err
		}
		_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
		return lastErr
	}

	if err := tx.TransitionTo(distsaga.StateCompensated); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_compensated", err)
	}

	e.logger.Infof("SAGA transaction [%s] compensated", tx.ID)
	return nil
}

// stepContext 创建步骤上下文
func (e *SagaExecutor) stepContext(ctx context.Context, stepTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := e.timeout
	if stepTimeout > 0 {
		timeout = stepTimeout
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
