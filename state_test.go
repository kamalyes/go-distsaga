/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 10:22:51
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 22:31:07
 * @FilePath: \go-distsaga\state_test.go
 * @Description: 事务状态和状态机测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== TxState 测试 ====================

func TestTxState_String(t *testing.T) {
	tests := []struct {
		state    TxState
		expected string
	}{
		{StatePending, "PENDING"},
		{StateRunning, "RUNNING"},
		{StateTrying, "TRYING"},
		{StateConfirming, "CONFIRMING"},
		{StateCanceling, "CANCELING"},
		{StateCompensating, "COMPENSATING"},
		{StatePreparing, "PREPARING"},
		{StatePrepared, "PREPARED"},
		{StateCommitting, "COMMITTING"},
		{StateAborting, "ABORTING"},
		{StateCommitted, "COMMITTED"},
		{StateCompensated, "COMPENSATED"},
		{StateRolledback, "ROLLEDBACK"},
		{StateFailed, "FAILED"},
		{StateSuspended, "SUSPENDED"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestTxState_IsFinal(t *testing.T) {
	t.Run("终态", func(t *testing.T) {
		assert.True(t, StateCommitted.IsFinal())
		assert.True(t, StateCompensated.IsFinal())
		assert.True(t, StateRolledback.IsFinal())
		assert.True(t, StateFailed.IsFinal())
	})

	t.Run("非终态", func(t *testing.T) {
		assert.False(t, StatePending.IsFinal())
		assert.False(t, StateRunning.IsFinal())
		assert.False(t, StateTrying.IsFinal())
		assert.False(t, StateCompensating.IsFinal())
		assert.False(t, StateSuspended.IsFinal())
	})
}

// ==================== TxMode 测试 ====================

func TestTxMode_String(t *testing.T) {
	tests := []struct {
		mode     TxMode
		expected string
	}{
		{ModeSaga, "SAGA"},
		{ModeTCC, "TCC"},
		{ModeXA, "XA"},
		{ModeWorkflow, "WORKFLOW"},
		{ModeOutbox, "OUTBOX"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.String())
		})
	}
}

// ==================== 状态转换测试 ====================

func TestCanTransition_SAGA(t *testing.T) {
	t.Run("合法转换", func(t *testing.T) {
		assert.True(t, CanTransition(ModeSaga, StatePending, StateRunning))
		assert.True(t, CanTransition(ModeSaga, StateRunning, StateCommitted))
		assert.True(t, CanTransition(ModeSaga, StateRunning, StateCompensating))
		assert.True(t, CanTransition(ModeSaga, StateCompensating, StateCompensated))
	})

	t.Run("非法转换", func(t *testing.T) {
		assert.False(t, CanTransition(ModeSaga, StatePending, StateCommitted))
		assert.False(t, CanTransition(ModeSaga, StateCommitted, StateRunning))
		assert.False(t, CanTransition(ModeSaga, StateCompensated, StateRunning))
	})
}

func TestCanTransition_TCC(t *testing.T) {
	t.Run("合法转换", func(t *testing.T) {
		assert.True(t, CanTransition(ModeTCC, StatePending, StateTrying))
		assert.True(t, CanTransition(ModeTCC, StateTrying, StateConfirming))
		assert.True(t, CanTransition(ModeTCC, StateTrying, StateCanceling))
		assert.True(t, CanTransition(ModeTCC, StateConfirming, StateCommitted))
		assert.True(t, CanTransition(ModeTCC, StateCanceling, StateRolledback))
	})

	t.Run("非法转换", func(t *testing.T) {
		assert.False(t, CanTransition(ModeTCC, StatePending, StateCommitted))
		assert.False(t, CanTransition(ModeTCC, StateTrying, StateCommitted))
	})
}

func TestCanTransition_XA(t *testing.T) {
	t.Run("合法转换", func(t *testing.T) {
		assert.True(t, CanTransition(ModeXA, StatePending, StatePreparing))
		assert.True(t, CanTransition(ModeXA, StatePreparing, StatePrepared))
		assert.True(t, CanTransition(ModeXA, StatePrepared, StateCommitting))
		assert.True(t, CanTransition(ModeXA, StateCommitting, StateCommitted))
		assert.True(t, CanTransition(ModeXA, StatePrepared, StateAborting))
		assert.True(t, CanTransition(ModeXA, StateAborting, StateRolledback))
	})

	t.Run("非法转换", func(t *testing.T) {
		assert.False(t, CanTransition(ModeXA, StatePending, StateCommitted))
		assert.False(t, CanTransition(ModeXA, StatePreparing, StateCommitted))
	})
}

func TestCanTransition_Outbox(t *testing.T) {
	t.Run("合法转换", func(t *testing.T) {
		assert.True(t, CanTransition(ModeOutbox, StatePending, StateSubmitting))
		assert.True(t, CanTransition(ModeOutbox, StateSubmitting, StateSubmitted))
		assert.True(t, CanTransition(ModeOutbox, StateSubmitted, StateCommitted))
	})

	t.Run("非法转换", func(t *testing.T) {
		assert.False(t, CanTransition(ModeOutbox, StatePending, StateCommitted))
	})
}
