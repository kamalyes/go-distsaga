/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-28 11:55:18
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 23:08:55
 * @FilePath: \go-distsaga\step_test.go
 * @Description: 跨模块共享类型测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepResult(t *testing.T) {
	t.Run("创建带数据的 StepResult", func(t *testing.T) {
		r := StepResult{Data: map[string]string{"key": "value"}, Error: nil}
		assert.Equal(t, "value", r.Data["key"])
		assert.Nil(t, r.Error)
	})

	t.Run("创建带错误的 StepResult", func(t *testing.T) {
		r := StepResult{Error: errors.New("failed")}
		assert.NotNil(t, r.Error)
	})
}

func TestStepStateItem(t *testing.T) {
	t.Run("创建 StepStateItem", func(t *testing.T) {
		item := StepStateItem{
			Name:   "step1",
			Status: StateCommitted,
			Result: StepResult{Data: map[string]string{"k": "v"}},
		}
		assert.Equal(t, "step1", item.Name)
		assert.Equal(t, StateCommitted, item.Status)
		assert.Equal(t, "v", item.Result.Data["k"])
	})
}

func TestTCCBranchStateItem(t *testing.T) {
	t.Run("创建 TCCBranchStateItem", func(t *testing.T) {
		item := TCCBranchStateItem{
			Name:   "branch1",
			Status: StatePrepared,
			TryResult: StepResult{Data: map[string]string{"reserved": "100"}},
		}
		assert.Equal(t, "branch1", item.Name)
		assert.Equal(t, StatePrepared, item.Status)
		assert.Equal(t, "100", item.TryResult.Data["reserved"])
	})
}

func TestXABranchStateItem(t *testing.T) {
	t.Run("创建 XABranchStateItem", func(t *testing.T) {
		item := XABranchStateItem{
			Name:   "xa-branch1",
			Status: StatePrepared,
		}
		assert.Equal(t, "xa-branch1", item.Name)
		assert.Equal(t, StatePrepared, item.Status)
	})
}

func TestOutboxMessageStateItem(t *testing.T) {
	t.Run("创建 OutboxMessageStateItem", func(t *testing.T) {
		item := OutboxMessageStateItem{
			ID:        "msg-001",
			Status:    StateCommitted,
			Target:    "order-service",
			Confirmed: true,
			Retries:   0,
		}
		assert.Equal(t, "msg-001", item.ID)
		assert.Equal(t, StateCommitted, item.Status)
		assert.Equal(t, "order-service", item.Target)
		assert.True(t, item.Confirmed)
		assert.Equal(t, 0, item.Retries)
	})
}
