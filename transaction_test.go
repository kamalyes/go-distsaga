/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-29 11:08:55
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:59:08
 * @FilePath: \go-distsaga\transaction_test.go
 * @Description: Transaction 核心数据模型测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	t.Run("创建 SAGA 事务", func(t *testing.T) {
		tx := NewTransaction("test-saga", ModeSaga)
		require.NotNil(t, tx)
		assert.NotEmpty(t, tx.ID)
		assert.Contains(t, tx.ID, "distsaga-")
		assert.Equal(t, "test-saga", tx.Name)
		assert.Equal(t, ModeSaga, tx.Mode)
		assert.Equal(t, StatePending, tx.State)
		assert.False(t, tx.CreatedAt.IsZero())
		assert.False(t, tx.UpdatedAt.IsZero())
		assert.Nil(t, tx.FinishedAt)
	})

	t.Run("创建 TCC 事务", func(t *testing.T) {
		tx := NewTransaction("test-tcc", ModeTCC)
		assert.Equal(t, ModeTCC, tx.Mode)
		assert.Equal(t, StatePending, tx.State)
	})

	t.Run("创建 XA 事务", func(t *testing.T) {
		tx := NewTransaction("test-xa", ModeXA)
		assert.Equal(t, ModeXA, tx.Mode)
	})
}

func TestTransaction_TransitionTo(t *testing.T) {
	t.Run("SAGA 合法状态转换", func(t *testing.T) {
		tx := NewTransaction("saga", ModeSaga)
		assert.Equal(t, StatePending, tx.State)

		err := tx.TransitionTo(StateRunning)
		assert.NoError(t, err)
		assert.Equal(t, StateRunning, tx.State)

		err = tx.TransitionTo(StateCommitted)
		assert.NoError(t, err)
		assert.Equal(t, StateCommitted, tx.State)
		assert.NotNil(t, tx.FinishedAt)
	})

	t.Run("SAGA 非法状态转换", func(t *testing.T) {
		tx := NewTransaction("saga", ModeSaga)
		err := tx.TransitionTo(StateCommitted)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrStateTransition)
		assert.Equal(t, StatePending, tx.State)
	})

	t.Run("TCC 合法状态转换", func(t *testing.T) {
		tx := NewTransaction("tcc", ModeTCC)

		err := tx.TransitionTo(StateTrying)
		assert.NoError(t, err)

		err = tx.TransitionTo(StateConfirming)
		assert.NoError(t, err)

		err = tx.TransitionTo(StateCommitted)
		assert.NoError(t, err)
		assert.True(t, tx.IsFinal())
	})

	t.Run("XA 合法状态转换", func(t *testing.T) {
		tx := NewTransaction("xa", ModeXA)

		err := tx.TransitionTo(StatePreparing)
		assert.NoError(t, err)

		err = tx.TransitionTo(StatePrepared)
		assert.NoError(t, err)

		err = tx.TransitionTo(StateCommitting)
		assert.NoError(t, err)

		err = tx.TransitionTo(StateCommitted)
		assert.NoError(t, err)
	})

	t.Run("终态设置 FinishedAt", func(t *testing.T) {
		tx := NewTransaction("test", ModeSaga)
		assert.Nil(t, tx.FinishedAt)

		tx.TransitionTo(StateRunning)
		assert.Nil(t, tx.FinishedAt)

		tx.TransitionTo(StateCommitted)
		assert.NotNil(t, tx.FinishedAt)
	})
}

func TestTransaction_IsFinal(t *testing.T) {
	tests := []struct {
		name     string
		state    TxState
		expected bool
	}{
		{"PENDING 非终态", StatePending, false},
		{"RUNNING 非终态", StateRunning, false},
		{"COMMITTED 终态", StateCommitted, true},
		{"COMPENSATED 终态", StateCompensated, true},
		{"ROLLEDBACK 终态", StateRolledback, true},
		{"FAILED 终态", StateFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{State: tt.state}
			assert.Equal(t, tt.expected, tx.IsFinal())
		})
	}
}

func TestTransaction_GetStepResult(t *testing.T) {
	t.Run("获取存在的步骤结果", func(t *testing.T) {
		tx := &Transaction{
			Steps: []StepStateItem{
				{Name: "step1", Result: StepResult{Data: map[string]string{"key": "value1"}}},
				{Name: "step2", Result: StepResult{Data: map[string]string{"key": "value2"}}},
			},
		}

		result, ok := tx.GetStepResult("step1")
		assert.True(t, ok)
		assert.Equal(t, "value1", result.Data["key"])
	})

	t.Run("获取不存在的步骤结果", func(t *testing.T) {
		tx := &Transaction{Steps: []StepStateItem{}}
		_, ok := tx.GetStepResult("nonexistent")
		assert.False(t, ok)
	})
}

func TestTransaction_GetTCCBranchResult(t *testing.T) {
	t.Run("获取存在的分支结果", func(t *testing.T) {
		tx := &Transaction{
			TCCBranches: []TCCBranchStateItem{
				{Name: "branch1", TryResult: StepResult{Data: map[string]string{"key": "value1"}}},
			},
		}

		result, ok := tx.GetTCCBranchResult("branch1")
		assert.True(t, ok)
		assert.Equal(t, "value1", result.Data["key"])
	})

	t.Run("获取不存在的分支结果", func(t *testing.T) {
		tx := &Transaction{TCCBranches: []TCCBranchStateItem{}}
		_, ok := tx.GetTCCBranchResult("nonexistent")
		assert.False(t, ok)
	})
}
