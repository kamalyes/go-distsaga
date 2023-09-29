/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 10:22:08
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-27 18:05:33
 * @FilePath: \go-distsaga\errors.go
 * @Description: 分布式事务错误定义 - 统一错误码和错误类型
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"errors"
	"fmt"
)

// ==================== 标准错误定义 ====================

var (
	ErrTransactionNotFound    = errors.New("distsaga: transaction not found")         // 事务不存在
	ErrTransactionAlreadyDone = errors.New("distsaga: transaction already completed") // 事务已完成
	ErrTransactionCancelled   = errors.New("distsaga: transaction cancelled")         // 事务已取消
	ErrTransactionTimeout     = errors.New("distsaga: transaction timeout")           // 事务超时
	ErrInvalidStep            = errors.New("distsaga: invalid step")                  // 无效步骤
	ErrInvalidBranch          = errors.New("distsaga: invalid branch")                // 无效分支
	ErrInvalidAdapter         = errors.New("distsaga: adapter is nil")                // 适配器为空
	ErrInvalidTransactionID   = errors.New("distsaga: invalid transaction id")        // 无效事务 ID
	ErrCompensationFailed     = errors.New("distsaga: compensation failed")           // 补偿失败
	ErrConfirmFailed          = errors.New("distsaga: confirm failed")                // 确认失败
	ErrCancelFailed           = errors.New("distsaga: cancel failed")                 // 取消失败
	ErrPrepareFailed          = errors.New("distsaga: prepare failed")                // 准备失败
	ErrCommitFailed           = errors.New("distsaga: commit failed")                 // 提交失败
	ErrRollbackFailed         = errors.New("distsaga: rollback failed")               // 回滚失败
	ErrStateTransition        = errors.New("distsaga: invalid state transition")      // 非法状态转换
	ErrAdapterOperation       = errors.New("distsaga: adapter operation failed")      // 适配器操作失败
	ErrRecoveryFailed         = errors.New("distsaga: recovery failed")               // 恢复失败
	ErrDuplicateOperation     = errors.New("distsaga: duplicate operation")           // 重复操作
	ErrBranchBarrier          = errors.New("distsaga: branch barrier rejected")       // 分支屏障拒绝（幂等控制）
)

// ==================== 事务错误类型 ====================

// TransactionError 事务错误
// 携带事务 ID、步骤名称、模式、状态等上下文信息
type TransactionError struct {
	TxID  string  // 事务 ID
	Step  string  // 步骤名称
	Mode  TxMode  // 事务模式
	State TxState // 事务状态
	Err   error   // 原始错误
	Msg   string  // 错误描述
}

// NewTransactionError 创建事务错误
func NewTransactionError(txID, step string, mode TxMode, state TxState, err error, msg string) *TransactionError {
	return &TransactionError{TxID: txID, Step: step, Mode: mode, State: state, Err: err, Msg: msg}
}

// Error 返回错误字符串
func (e *TransactionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("distsaga: [%s] step=%s mode=%s state=%s: %s: %v", e.TxID, e.Step, e.Mode, e.State, e.Msg, e.Err)
	}
	return fmt.Sprintf("distsaga: [%s] step=%s mode=%s state=%s: %s", e.TxID, e.Step, e.Mode, e.State, e.Msg)
}

// Unwrap 返回原始错误，支持 errors.Is / errors.As
func (e *TransactionError) Unwrap() error {
	return e.Err
}

// WrapAdapterError 包装适配器操作错误
// 统一添加 ErrAdapterOperation 前缀和操作名称
func WrapAdapterError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", ErrAdapterOperation, op, err)
}
