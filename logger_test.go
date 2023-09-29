/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-25 13:19:27
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-28 10:31:55
 * @FilePath: \go-distsaga\logger_test.go
 * @Description: 日志配置测试
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package distsaga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultLogger(t *testing.T) {
	t.Run("创建默认日志记录器", func(t *testing.T) {
		logger := NewDefaultLogger()
		assert.NotNil(t, logger)
	})
}
