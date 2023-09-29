/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 15:27:08
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:52:28
 * @FilePath: \go-distsaga\tcc\executor.go
 * @Description: TCC 执行器 - Try / Confirm / Cancel 三阶段提交
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package tcc

import (
	"context"
	"fmt"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// TCCExecutor TCC 事务执行器
type TCCExecutor struct {
	adapter distsaga.StoreAdapter
	logger  logger.ILogger
	retry   *retry.Retry
	timeout time.Duration
}

// NewExecutor 创建 TCC 执行器
func NewExecutor(adapter distsaga.StoreAdapter, l logger.ILogger, r *retry.Retry, timeout time.Duration) *TCCExecutor {
	return &TCCExecutor{
		adapter: adapter,
		logger:  l,
		retry:   r,
		timeout: timeout,
	}
}

// Execute 执行 TCC 事务
func (e *TCCExecutor) Execute(ctx context.Context, tx *distsaga.Transaction, branches []*Branch) (*distsaga.Transaction, error) {
	if err := tx.TransitionTo(distsaga.StateTrying); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_trying", err)
	}

	var triedBranches []int
	for i, branch := range branches {
		branchCtx, cancel := e.branchContext(ctx, branch.Timeout)

		var result distsaga.StepResult
		var err error

		if e.retry != nil && branch.Retries > 0 {
			r := retry.NewRetryWithCtx(branchCtx).
				SetAttemptCount(branch.Retries + 1).
				SetInterval(100 * time.Millisecond)
			err = r.Do(func() error {
				var tryErr error
				result, tryErr = branch.Try(branchCtx)
				return tryErr
			})
		} else {
			result, err = branch.Try(branchCtx)
		}
		cancel()

		bs := BranchState{
			Name:      branch.Name,
			StartedAt: time.Now(),
		}

		if err != nil {
			e.logger.Errorf("TCC branch [%s] try failed: %v", branch.Name, err)
			bs.Status = distsaga.StateFailed
			bs.TryResult = distsaga.StepResult{Error: err}
			tx.TCCBranches = append(tx.TCCBranches, distsaga.TCCBranchStateItem{
				Name:      bs.Name,
				Status:    bs.Status,
				TryResult: bs.TryResult,
				StartedAt: bs.StartedAt,
				EndedAt:   bs.EndedAt,
				Retries:   bs.Retries,
			})

			if cancelErr := e.cancel(ctx, tx, branches, triedBranches); cancelErr != nil {
				return tx, fmt.Errorf("%w: branch=%s cancel=%v", distsaga.ErrCancelFailed, branch.Name, cancelErr)
			}
			return tx, distsaga.NewTransactionError(tx.ID, branch.Name, distsaga.ModeTCC, tx.State, err, "try failed, cancelled")
		}

		bs.Status = distsaga.StatePrepared
		bs.TryResult = result
		bs.EndedAt = time.Now()
		tx.TCCBranches = append(tx.TCCBranches, distsaga.TCCBranchStateItem{
			Name:      bs.Name,
			Status:    bs.Status,
			TryResult: bs.TryResult,
			StartedAt: bs.StartedAt,
			EndedAt:   bs.EndedAt,
			Retries:   bs.Retries,
		})
		triedBranches = append(triedBranches, i)

		e.logger.Infof("TCC branch [%s] try succeeded", branch.Name)
	}

	if err := e.confirm(ctx, tx, branches, triedBranches); err != nil {
		return tx, err
	}

	return tx, nil
}

// confirm 执行 Confirm 阶段
func (e *TCCExecutor) confirm(ctx context.Context, tx *distsaga.Transaction, branches []*Branch, triedIndices []int) error {
	if err := tx.TransitionTo(distsaga.StateConfirming); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_confirming", err)
	}

	for _, idx := range triedIndices {
		branch := branches[idx]
		result := tx.TCCBranches[idx].TryResult

		var err error
		if e.retry != nil {
			err = e.retry.Do(func() error {
				return branch.Confirm(ctx, result)
			})
		} else {
			err = branch.Confirm(ctx, result)
		}

		if err != nil {
			e.logger.Errorf("TCC branch [%s] confirm failed: %v", branch.Name, err)
			if suspendErr := tx.TransitionTo(distsaga.StateSuspended); suspendErr != nil {
				return suspendErr
			}
			_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
			return fmt.Errorf("%w: branch=%s", distsaga.ErrConfirmFailed, branch.Name)
		}

		tx.TCCBranches[idx].Status = distsaga.StateCommitted
		e.logger.Infof("TCC branch [%s] confirmed", branch.Name)
	}

	if err := tx.TransitionTo(distsaga.StateCommitted); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_committed", err)
	}

	e.logger.Infof("TCC transaction [%s] committed", tx.ID)
	return nil
}

// cancel 执行 Cancel 阶段
func (e *TCCExecutor) cancel(ctx context.Context, tx *distsaga.Transaction, branches []*Branch, triedIndices []int) error {
	if err := tx.TransitionTo(distsaga.StateCanceling); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_canceling", err)
	}

	var lastErr error
	for i := len(triedIndices) - 1; i >= 0; i-- {
		idx := triedIndices[i]
		branch := branches[idx]
		result := tx.TCCBranches[idx].TryResult

		var cancelErr error
		if e.retry != nil {
			cancelErr = e.retry.Do(func() error {
				return branch.Cancel(ctx, result)
			})
		} else {
			cancelErr = branch.Cancel(ctx, result)
		}

		if cancelErr != nil {
			e.logger.Errorf("TCC branch [%s] cancel failed: %v", branch.Name, cancelErr)
			tx.TCCBranches[idx].Status = distsaga.StateFailed
			lastErr = cancelErr
		} else {
			tx.TCCBranches[idx].Status = distsaga.StateRolledback
			e.logger.Infof("TCC branch [%s] cancelled", branch.Name)
		}
	}

	if lastErr != nil {
		if err := tx.TransitionTo(distsaga.StateSuspended); err != nil {
			return err
		}
		_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
		return lastErr
	}

	if err := tx.TransitionTo(distsaga.StateRolledback); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_rolledback", err)
	}

	e.logger.Infof("TCC transaction [%s] rolled back", tx.ID)
	return nil
}

// branchContext 创建分支上下文
func (e *TCCExecutor) branchContext(ctx context.Context, branchTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := e.timeout
	if branchTimeout > 0 {
		timeout = branchTimeout
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
