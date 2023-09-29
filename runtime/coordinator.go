/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 08:21:33
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 11:22:07
 * @FilePath: \go-distsaga\runtime\coordinator.go
 * @Description: Runtime 编排方法 - 驱动各事务模式的执行 + 事件通知
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package runtime

import (
	"context"
	"fmt"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-distsaga/outbox"
	"github.com/kamalyes/go-distsaga/saga"
	"github.com/kamalyes/go-distsaga/tcc"
	"github.com/kamalyes/go-distsaga/workflow"
	"github.com/kamalyes/go-distsaga/xa"
)

// Saga 执行 SAGA 事务
func (r *Runtime) Saga(ctx context.Context, name string, opts ...saga.Option) (*distsaga.Transaction, error) {
	o := saga.NewOptions(opts...)

	if len(o.Steps) == 0 {
		return nil, fmt.Errorf("%w: saga requires at least one step", distsaga.ErrInvalidStep)
	}

	tx := distsaga.NewTransaction(name, distsaga.ModeSaga)
	if o.StepTimeout > 0 {
		for _, step := range o.Steps {
			if step.Timeout == 0 {
				step.Timeout = o.StepTimeout
			}
		}
	}

	if err := r.adapter.SaveTransaction(ctx, tx); err != nil {
		return nil, distsaga.WrapAdapterError("save", err)
	}

	executor := saga.NewExecutor(r.adapter, r.logger, r.retry, r.stepTimeout)
	result, err := executor.Execute(ctx, tx, o.Steps)
	if err != nil {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionFailed, "")
		return result, err
	}

	if result.State == distsaga.StateCommitted {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCommitted, "")
	} else if result.State == distsaga.StateCompensated {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCompensated, "")
	}

	return result, nil
}

// TCC 执行 TCC 事务
func (r *Runtime) TCC(ctx context.Context, name string, opts ...tcc.Option) (*distsaga.Transaction, error) {
	o := tcc.NewOptions(opts...)

	if len(o.Branches) == 0 {
		return nil, fmt.Errorf("%w: tcc requires at least one branch", distsaga.ErrInvalidBranch)
	}

	tx := distsaga.NewTransaction(name, distsaga.ModeTCC)
	if o.StepTimeout > 0 {
		for _, branch := range o.Branches {
			if branch.Timeout == 0 {
				branch.Timeout = o.StepTimeout
			}
		}
	}

	if err := r.adapter.SaveTransaction(ctx, tx); err != nil {
		return nil, distsaga.WrapAdapterError("save", err)
	}

	executor := tcc.NewExecutor(r.adapter, r.logger, r.retry, r.stepTimeout)
	result, err := executor.Execute(ctx, tx, o.Branches)
	if err != nil {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionFailed, "")
		return result, err
	}

	if result.State == distsaga.StateCommitted {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCommitted, "")
	} else if result.State == distsaga.StateRolledback {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionRolledback, "")
	}

	return result, nil
}

// XA 执行 XA 事务
func (r *Runtime) XA(ctx context.Context, name string, opts ...xa.Option) (*distsaga.Transaction, error) {
	o := xa.NewOptions(opts...)

	if len(o.Branches) == 0 {
		return nil, fmt.Errorf("%w: xa requires at least one branch", distsaga.ErrInvalidBranch)
	}

	tx := distsaga.NewTransaction(name, distsaga.ModeXA)

	if err := r.adapter.SaveTransaction(ctx, tx); err != nil {
		return nil, distsaga.WrapAdapterError("save", err)
	}

	executor := xa.NewExecutor(r.adapter, r.logger, r.retry, r.stepTimeout)
	result, err := executor.Execute(ctx, tx, o.Branches)
	if err != nil {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionFailed, "")
		return result, err
	}

	if result.State == distsaga.StateCommitted {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCommitted, "")
	} else if result.State == distsaga.StateRolledback {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionRolledback, "")
	}

	return result, nil
}

// Workflow 执行 Workflow 事务
func (r *Runtime) Workflow(ctx context.Context, name string, opts ...workflow.Option) (*distsaga.Transaction, error) {
	o := workflow.NewOptions(opts...)

	if o.Handler == nil {
		return nil, fmt.Errorf("%w: workflow requires a handler", distsaga.ErrInvalidStep)
	}

	tx := distsaga.NewTransaction(name, distsaga.ModeWorkflow)

	if err := r.adapter.SaveTransaction(ctx, tx); err != nil {
		return nil, distsaga.WrapAdapterError("save", err)
	}

	executor := workflow.NewExecutor(r.adapter, r.logger, r.stepTimeout)
	result, err := executor.Execute(ctx, tx, o.Handler, o.Data)
	if err != nil {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionFailed, "")
		return result, err
	}

	if result.State == distsaga.StateCommitted {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCommitted, "")
	} else if result.State == distsaga.StateCompensated {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCompensated, "")
	}

	return result, nil
}

// Outbox 执行 Outbox 事务
func (r *Runtime) Outbox(ctx context.Context, name string, opts ...outbox.Option) (*distsaga.Transaction, error) {
	o := outbox.NewOptions(opts...)

	if len(o.Messages) == 0 {
		return nil, fmt.Errorf("%w: outbox requires at least one message", distsaga.ErrInvalidStep)
	}

	tx := distsaga.NewTransaction(name, distsaga.ModeOutbox)

	if err := r.adapter.SaveTransaction(ctx, tx); err != nil {
		return nil, distsaga.WrapAdapterError("save", err)
	}

	executor := outbox.NewExecutor(r.adapter, r.logger, r.retry, r.stepTimeout)
	result, err := executor.Execute(ctx, tx, o.Messages, o.BusinessOp)
	if err != nil {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionFailed, "")
		return result, err
	}

	if result.State == distsaga.StateCommitted {
		r.notifyEvent(ctx, tx, distsaga.EventTransactionCommitted, "")
	}

	return result, nil
}

// notifyEvent 发布事务事件
func (r *Runtime) notifyEvent(ctx context.Context, tx *distsaga.Transaction, eventType distsaga.TransactionEventType, stepName string) {
	if r.notifier == nil {
		return
	}

	event := &distsaga.TransactionEvent{
		TxID:      tx.ID,
		TxName:    tx.Name,
		Mode:      tx.Mode,
		EventType: eventType,
		State:     tx.State,
		StepName:  stepName,
	}

	if err := r.notifier.Publish(ctx, event); err != nil {
		r.logger.Errorf("Failed to publish event %s for transaction [%s]: %v", eventType, tx.ID, err)
	}
}
