/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 10:35:11
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 01:03:15
 * @FilePath: \go-distsaga\runtime\runtime_test.go
 * @Description: Runtime 运行时引擎测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package runtime

import (
	"context"
	"testing"
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
	"github.com/kamalyes/go-distsaga/outbox"
	"github.com/kamalyes/go-distsaga/saga"
	"github.com/kamalyes/go-distsaga/tcc"
	"github.com/kamalyes/go-distsaga/workflow"
	"github.com/kamalyes/go-distsaga/xa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultConfig(t *testing.T) {
	t.Run("默认配置创建", func(t *testing.T) {
		rt, err := New()
		require.NoError(t, err)
		assert.NotNil(t, rt)
	})
}

func TestNew_NilAdapter(t *testing.T) {
	t.Run("适配器为 nil 返回错误", func(t *testing.T) {
		rt, err := New(func(r *Runtime) { r.adapter = nil })
		assert.Error(t, err)
		assert.Nil(t, rt)
	})
}

func TestNew_WithCustomAdapter(t *testing.T) {
	t.Run("自定义适配器", func(t *testing.T) {
		adapter := distsaga.NewMemoryAdapter()
		rt, err := New(WithAdapter(adapter))
		require.NoError(t, err)
		assert.NotNil(t, rt)
	})
}

func TestNew_WithStepTimeout(t *testing.T) {
	t.Run("自定义步骤超时", func(t *testing.T) {
		rt, err := New(WithStepTimeout(10 * time.Second))
		require.NoError(t, err)
		assert.Equal(t, 10*time.Second, rt.stepTimeout)
	})
}

func TestNew_WithRecoveryInterval(t *testing.T) {
	t.Run("自定义恢复间隔", func(t *testing.T) {
		rt, err := New(WithRecoveryInterval(5 * time.Second))
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, rt.recoveryInterval)
	})
}

func TestNew_WithRecoveryMaxAge(t *testing.T) {
	t.Run("自定义恢复最大年龄", func(t *testing.T) {
		rt, err := New(WithRecoveryMaxAge(48 * time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 48*time.Hour, rt.recoveryMaxAge)
	})
}

func TestNew_WithTransactionTTL(t *testing.T) {
	t.Run("自定义事务 TTL", func(t *testing.T) {
		rt, err := New(WithTransactionTTL(24 * time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 24*time.Hour, rt.transactionTTL)
	})
}

func TestNew_WithEnableRecovery(t *testing.T) {
	t.Run("禁用恢复", func(t *testing.T) {
		rt, err := New(WithEnableRecovery(false))
		require.NoError(t, err)
		assert.Nil(t, rt.recovery)
	})
}

func TestRuntime_Saga(t *testing.T) {
	t.Run("执行 SAGA 事务", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		result, err := rt.Saga(ctx, "test-saga",
			saga.WithSteps(
				saga.NewStep("step1",
					func(ctx context.Context) (distsaga.StepResult, error) {
						return distsaga.StepResult{Data: map[string]string{"key": "val"}}, nil
					},
					func(ctx context.Context, result distsaga.StepResult) error { return nil },
				),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})

	t.Run("SAGA 无步骤返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		_, err := rt.Saga(ctx, "empty-saga")
		assert.Error(t, err)
	})
}

func TestRuntime_TCC(t *testing.T) {
	t.Run("执行 TCC 事务", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		result, err := rt.TCC(ctx, "test-tcc",
			tcc.WithBranches(
				tcc.NewBranch("branch1",
					func(ctx context.Context) (distsaga.StepResult, error) {
						return distsaga.StepResult{}, nil
					},
					func(ctx context.Context, result distsaga.StepResult) error { return nil },
					func(ctx context.Context, result distsaga.StepResult) error { return nil },
				),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})

	t.Run("TCC 无分支返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		_, err := rt.TCC(ctx, "empty-tcc")
		assert.Error(t, err)
	})
}

func TestRuntime_XA(t *testing.T) {
	t.Run("执行 XA 事务", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		result, err := rt.XA(ctx, "test-xa",
			xa.WithBranches(
				xa.NewBranch("branch1", &testResource{}),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})

	t.Run("XA 无分支返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		_, err := rt.XA(ctx, "empty-xa")
		assert.Error(t, err)
	})
}

func TestRuntime_Workflow(t *testing.T) {
	t.Run("执行 Workflow 事务", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		result, err := rt.Workflow(ctx, "test-workflow",
			workflow.WithHandler(func(wf *workflow.Workflow, data []byte) error {
				wf.OnConfirm(func(ctx context.Context, data []byte) error { return nil })
				return nil
			}),
		)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})

	t.Run("Workflow 无 Handler 返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		_, err := rt.Workflow(ctx, "empty-workflow")
		assert.Error(t, err)
	})
}

func TestRuntime_Outbox(t *testing.T) {
	t.Run("执行 Outbox 事务", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		result, err := rt.Outbox(ctx, "test-outbox",
			outbox.WithMessages(
				&outbox.Message{ID: "msg-001", Target: "https://example.com/api", Body: []byte(`{}`)},
			),
		)

		require.NoError(t, err)
		assert.Equal(t, distsaga.StateCommitted, result.State)
	})

	t.Run("Outbox 无消息返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		_, err := rt.Outbox(ctx, "empty-outbox")
		assert.Error(t, err)
	})
}

func TestRuntime_StartStopRecovery(t *testing.T) {
	t.Run("启动和停止恢复", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(true))
		ctx := context.Background()

		err := rt.StartRecovery(ctx)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		rt.StopRecovery()
	})

	t.Run("恢复器为 nil 时返回错误", func(t *testing.T) {
		rt, _ := New(WithEnableRecovery(false))
		ctx := context.Background()

		err := rt.StartRecovery(ctx)
		assert.Error(t, err)
	})
}

func TestRuntime_WithNotifier(t *testing.T) {
	t.Run("自定义通知器", func(t *testing.T) {
		notifier := distsaga.NewMemoryNotifier()
		rt, err := New(WithNotifier(notifier), WithEnableRecovery(false))
		require.NoError(t, err)
		assert.NotNil(t, rt.notifier)
	})
}

func TestRuntime_WithLogger(t *testing.T) {
	t.Run("自定义日志", func(t *testing.T) {
		logger := distsaga.NewDefaultLogger()
		rt, err := New(WithLogger(logger), WithEnableRecovery(false))
		require.NoError(t, err)
		assert.NotNil(t, rt.logger)
	})
}

type testResource struct{}

func (r *testResource) Prepare(ctx context.Context) error  { return nil }
func (r *testResource) Commit(ctx context.Context) error   { return nil }
func (r *testResource) Rollback(ctx context.Context) error { return nil }
