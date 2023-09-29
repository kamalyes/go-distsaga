/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-27 10:35:11
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 15:19:33
 * @FilePath: \go-distsaga\testutil.go
 * @Description: 测试辅助工具 - 供子包测试使用的公共函数
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"time"

	"github.com/kamalyes/go-toolbox/pkg/retry"
)

// NewTestRetry 创建用于测试的重试策略
func NewTestRetry() *retry.Retry {
	return retry.NewRetry().
		SetAttemptCount(3).
		SetInterval(10 * time.Millisecond)
}
