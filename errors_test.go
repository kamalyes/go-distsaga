/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 11:05:51
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 00:17:25
 * @FilePath: \go-distsaga\errors_test.go
 * @Description: 错误类型测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionError(t *testing.T) {
	t.Run("创建事务错误", func(t *testing.T) {
		err := NewTransactionError("tx-001", "step1", ModeSaga, StateRunning, errors.New("inner"), "something went wrong")
		assert.Equal(t, "tx-001", err.TxID)
		assert.Equal(t, "step1", err.Step)
		assert.Equal(t, ModeSaga, err.Mode)
		assert.Equal(t, StateRunning, err.State)
		assert.Equal(t, "something went wrong", err.Msg)
	})

	t.Run("Error 方法输出", func(t *testing.T) {
		err := NewTransactionError("tx-001", "step1", ModeSaga, StateRunning, errors.New("inner"), "test error")
		msg := err.Error()
		assert.Contains(t, msg, "tx-001")
		assert.Contains(t, msg, "step1")
		assert.Contains(t, msg, "SAGA")
		assert.Contains(t, msg, "test error")
	})

	t.Run("Unwrap 返回原始错误", func(t *testing.T) {
		inner := errors.New("inner error")
		err := NewTransactionError("tx-001", "step1", ModeSaga, StateRunning, inner, "test")
		assert.Equal(t, inner, err.Unwrap())
	})
}

func TestWrapAdapterError(t *testing.T) {
	t.Run("包装非空错误", func(t *testing.T) {
		inner := errors.New("connection refused")
		wrapped := WrapAdapterError("save", inner)
		assert.ErrorIs(t, wrapped, ErrAdapterOperation)
		assert.Contains(t, wrapped.Error(), "save")
		assert.Contains(t, wrapped.Error(), "connection refused")
	})

	t.Run("包装空错误返回 nil", func(t *testing.T) {
		result := WrapAdapterError("save", nil)
		assert.Nil(t, result)
	})
}

func TestPredefinedErrors(t *testing.T) {
	t.Run("预定义错误存在", func(t *testing.T) {
		assert.NotNil(t, ErrStateTransition)
		assert.NotNil(t, ErrTransactionNotFound)
		assert.NotNil(t, ErrDuplicateOperation)
		assert.NotNil(t, ErrAdapterOperation)
		assert.NotNil(t, ErrCompensationFailed)
		assert.NotNil(t, ErrCancelFailed)
		assert.NotNil(t, ErrConfirmFailed)
		assert.NotNil(t, ErrPrepareFailed)
		assert.NotNil(t, ErrTransactionTimeout)
		assert.NotNil(t, ErrRollbackFailed)
		assert.NotNil(t, ErrTransactionAlreadyDone)
	})
}
