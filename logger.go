/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 13:19:27
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-06-02 00:57:05
 * @FilePath: \go-distsaga\logger.go
 * @Description: go-distsaga 日志配置 - 直接复用 go-logger
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"time"

	"github.com/kamalyes/go-logger"
)

// NewDefaultLogger 创建默认日志记录器
// 日志级别 INFO，前缀 [DISTSAGA]，关闭调用者显示，启用彩色输出
func NewDefaultLogger() logger.ILogger {
	return logger.NewLogger().
		WithLevel(logger.INFO).
		WithPrefix("[DISTSAGA] ").
		WithShowCaller(false).
		WithColorful(true).
		WithTimeFormat(time.DateTime)
}
