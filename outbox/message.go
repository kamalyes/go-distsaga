/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2023-09-26 11:17:05
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2023-09-29 08:17:05
 * @FilePath: \go-distsaga\outbox\message.go
 * @Description: Outbox 消息定义 - 两阶段消息（Better Outbox）
 *
 * Copyright (c) 2023 by kamalyes, All Rights Reserved.
 */
package outbox

import (
	"time"

	distsaga "github.com/kamalyes/go-distsaga"
)

// Message Outbox 消息定义
// 用于两阶段消息模式，确保业务操作和消息发送的最终一致性
type Message struct {
	ID       string            // 消息 ID
	Target   string            // 消息目标
	Body     []byte            // 消息体
	Headers  map[string]string // 消息头
	Delay    time.Duration     // 延迟发送时间
	Priority int               // 优先级
}

// MessageState Outbox 消息执行状态
type MessageState struct {
	ID        string           // 消息 ID
	Status    distsaga.TxState // 消息状态
	Target    string           // 消息目标
	SentAt    time.Time        // 发送时间
	Confirmed bool             // 是否已确认
	Retries   int              // 已重试次数
}
