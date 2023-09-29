/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 18:19:07
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 00:16:15
 * @FilePath: \go-distsaga\xa\executor.go
 * @Description: XA 执行器 - Prepare / Commit / Rollback 两阶段提交
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package xa

import (
	"context"
	"fmt"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// XAExecutor XA 事务执行器
type XAExecutor struct {
	adapter distsaga.StoreAdapter
	logger  logger.ILogger
	retry   *retry.Retry
	timeout time.Duration
}

// NewExecutor 创建 XA 执行器
func NewExecutor(adapter distsaga.StoreAdapter, l logger.ILogger, r *retry.Retry, timeout time.Duration) *XAExecutor {
	return &XAExecutor{
		adapter: adapter,
		logger:  l,
		retry:   r,
		timeout: timeout,
	}
}

// Execute 执行 XA 事务
func (e *XAExecutor) Execute(ctx context.Context, tx *distsaga.Transaction, branches []*Branch) (*distsaga.Transaction, error) {
	if err := tx.TransitionTo(distsaga.StatePreparing); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_preparing", err)
	}

	var preparedBranches []int
	for i, branch := range branches {
		branchCtx, cancel := e.branchContext(ctx, branch.Timeout)

		var err error
		if e.retry != nil {
			r := retry.NewRetryWithCtx(branchCtx).
				SetAttemptCount(3).
				SetInterval(100 * time.Millisecond)
			err = r.Do(func() error {
				return branch.Resource.Prepare(branchCtx)
			})
		} else {
			err = branch.Resource.Prepare(branchCtx)
		}
		cancel()

		bs := BranchState{
			Name:      branch.Name,
			StartedAt: time.Now(),
		}

		if err != nil {
			e.logger.Errorf("XA branch [%s] prepare failed: %v", branch.Name, err)
			bs.Status = distsaga.StateFailed
			tx.XABranches = append(tx.XABranches, distsaga.XABranchStateItem{
				Name:      bs.Name,
				Status:    bs.Status,
				StartedAt: bs.StartedAt,
				EndedAt:   bs.EndedAt,
			})

			if abortErr := e.abort(ctx, tx, branches, preparedBranches); abortErr != nil {
				return tx, fmt.Errorf("%w: branch=%s abort=%v", distsaga.ErrRollbackFailed, branch.Name, abortErr)
			}
			return tx, distsaga.NewTransactionError(tx.ID, branch.Name, distsaga.ModeXA, tx.State, err, "prepare failed, aborted")
		}

		bs.Status = distsaga.StatePrepared
		bs.EndedAt = time.Now()
		tx.XABranches = append(tx.XABranches, distsaga.XABranchStateItem{
			Name:      bs.Name,
			Status:    bs.Status,
			StartedAt: bs.StartedAt,
			EndedAt:   bs.EndedAt,
		})
		preparedBranches = append(preparedBranches, i)

		e.logger.Infof("XA branch [%s] prepared", branch.Name)
	}

	if err := tx.TransitionTo(distsaga.StatePrepared); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_prepared", err)
	}

	if err := e.commit(ctx, tx, branches, preparedBranches); err != nil {
		return tx, err
	}

	return tx, nil
}

// commit 执行 Commit 阶段
func (e *XAExecutor) commit(ctx context.Context, tx *distsaga.Transaction, branches []*Branch, preparedIndices []int) error {
	if err := tx.TransitionTo(distsaga.StateCommitting); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_committing", err)
	}

	for _, idx := range preparedIndices {
		branch := branches[idx]

		var err error
		if e.retry != nil {
			err = e.retry.Do(func() error {
				return branch.Resource.Commit(ctx)
			})
		} else {
			err = branch.Resource.Commit(ctx)
		}

		if err != nil {
			e.logger.Errorf("XA branch [%s] commit failed: %v", branch.Name, err)
			tx.XABranches[idx].Status = distsaga.StateFailed
			if suspendErr := tx.TransitionTo(distsaga.StateSuspended); suspendErr != nil {
				return suspendErr
			}
			_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
			return fmt.Errorf("%w: branch=%s", distsaga.ErrCommitFailed, branch.Name)
		}

		tx.XABranches[idx].Status = distsaga.StateCommitted
		e.logger.Infof("XA branch [%s] committed", branch.Name)
	}

	if err := tx.TransitionTo(distsaga.StateCommitted); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_committed", err)
	}

	e.logger.Infof("XA transaction [%s] committed", tx.ID)
	return nil
}

// abort 执行 Rollback 阶段
func (e *XAExecutor) abort(ctx context.Context, tx *distsaga.Transaction, branches []*Branch, preparedIndices []int) error {
	if err := tx.TransitionTo(distsaga.StateAborting); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_aborting", err)
	}

	var lastErr error
	for i := len(preparedIndices) - 1; i >= 0; i-- {
		idx := preparedIndices[i]
		branch := branches[idx]

		var rollbackErr error
		if e.retry != nil {
			rollbackErr = e.retry.Do(func() error {
				return branch.Resource.Rollback(ctx)
			})
		} else {
			rollbackErr = branch.Resource.Rollback(ctx)
		}

		if rollbackErr != nil {
			e.logger.Errorf("XA branch [%s] rollback failed: %v", branch.Name, rollbackErr)
			tx.XABranches[idx].Status = distsaga.StateFailed
			lastErr = rollbackErr
		} else {
			tx.XABranches[idx].Status = distsaga.StateRolledback
			e.logger.Infof("XA branch [%s] rolled back", branch.Name)
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

	e.logger.Infof("XA transaction [%s] rolled back", tx.ID)
	return nil
}

// branchContext 创建分支上下文
func (e *XAExecutor) branchContext(ctx context.Context, branchTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := e.timeout
	if branchTimeout > 0 {
		timeout = branchTimeout
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
