/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 15:22:09
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:33
 * @FilePath: \go-distsaga\workflow\executor.go
 * @Description: Workflow 执行器 - 灵活编排，可混合 SAGA/TCC/XA
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package workflow

import (
	"context"
	"fmt"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-logger"
)

// WorkflowExecutor Workflow 事务执行器
// 执行用户自定义的 handler，成功则确认所有分支，失败则逆序回滚
type WorkflowExecutor struct {
	adapter distsaga.StoreAdapter // 存储适配器
	logger  logger.ILogger        // 日志记录器
	timeout time.Duration         // 默认超时时间
}

// NewExecutor 创建 Workflow 执行器
func NewExecutor(adapter distsaga.StoreAdapter, l logger.ILogger, timeout time.Duration) *WorkflowExecutor {
	return &WorkflowExecutor{
		adapter: adapter,
		logger:  l,
		timeout: timeout,
	}
}

// Execute 执行 Workflow 事务
// 1. 状态转换 PENDING → RUNNING
// 2. 执行用户自定义的 handler（handler 中注册分支）
// 3. handler 成功，执行所有分支的 OnConfirm
// 4. handler 失败，逆序执行所有分支的 OnRollback / OnCancel
func (e *WorkflowExecutor) Execute(ctx context.Context, tx *distsaga.Transaction, handler func(wf *Workflow, data []byte) error, data []byte) (*distsaga.Transaction, error) {
	if err := tx.TransitionTo(distsaga.StateRunning); err != nil {
		return nil, err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return nil, distsaga.WrapAdapterError("update_running", err)
	}

	wf := &Workflow{
		ID:      tx.ID,
		Context: ctx,
		logger:  e.logger,
	}

	if err := handler(wf, data); err != nil {
		e.logger.Errorf("Workflow [%s] execution failed: %v", tx.ID, err)

		if rollbackErr := e.rollback(ctx, tx, wf); rollbackErr != nil {
			return tx, fmt.Errorf("%w: execute=%v rollback=%v", distsaga.ErrRollbackFailed, err, rollbackErr)
		}
		return tx, distsaga.NewTransactionError(tx.ID, "", distsaga.ModeWorkflow, tx.State, err, "workflow failed, rolled back")
	}

	if err := e.confirm(ctx, tx, wf); err != nil {
		return tx, err
	}

	return tx, nil
}

// confirm 执行所有分支的 OnConfirm 回调
func (e *WorkflowExecutor) confirm(ctx context.Context, tx *distsaga.Transaction, wf *Workflow) error {
	for _, branch := range wf.GetBranches() {
		if branch.OnConfirm != nil {
			if err := branch.OnConfirm(ctx, nil); err != nil {
				e.logger.Errorf("Workflow branch confirm failed: %v", err)
				if suspendErr := tx.TransitionTo(distsaga.StateSuspended); suspendErr != nil {
					return suspendErr
				}
				_ = e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil)
				return fmt.Errorf("%w: %v", distsaga.ErrConfirmFailed, err)
			}
		}
	}

	if err := tx.TransitionTo(distsaga.StateCommitted); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_committed", err)
	}

	e.logger.Infof("Workflow transaction [%s] committed", tx.ID)
	return nil
}

// rollback 逆序执行所有分支的 OnRollback / OnCancel 回调
// 优先使用 OnRollback，不存在则使用 OnCancel
func (e *WorkflowExecutor) rollback(ctx context.Context, tx *distsaga.Transaction, wf *Workflow) error {
	if err := tx.TransitionTo(distsaga.StateCompensating); err != nil {
		return err
	}
	if err := e.adapter.UpdateTransactionState(ctx, tx.ID, tx.State, nil); err != nil {
		return distsaga.WrapAdapterError("update_compensating", err)
	}

	branches := wf.GetBranches()
	var lastErr error
	for i := len(branches) - 1; i >= 0; i-- {
		branch := branches[i]
		rollbackFn := branch.OnRollback
		if rollbackFn == nil {
			rollbackFn = branch.OnCancel
		}
		if rollbackFn == nil {
			continue
		}

		if err := rollbackFn(ctx, nil); err != nil {
			e.logger.Errorf("Workflow branch rollback failed: %v", err)
			lastErr = err
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

	e.logger.Infof("Workflow transaction [%s] compensated at %s", tx.ID, time.Now().Format(time.DateTime))
	return nil
}
