// Package event_bus 提供事件总线功能
package event_bus

import (
	"fmt"
)

// EventBus 事件总线结构
type EventBus struct {
	subscribers map[string][]func(interface{})
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(interface{})),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType string, handler func(interface{})) {
	if eb.subscribers == nil {
		eb.subscribers = make(map[string][]func(interface{}))
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
	fmt.Printf("事件 %s 订阅完成\n", eventType)
}

// Publish 发布事件
func (eb *EventBus) Publish(eventType string, data interface{}) {
	if handlers, exists := eb.subscribers[eventType]; exists {
		for _, handler := range handlers {
			go handler(data) // 异步处理
		}
		fmt.Printf("事件 %s 发布完成，通知了 %d 个订阅者\n", eventType, len(handlers))
	}
}

// Cleanup 清理事件总线资源
func (eb *EventBus) Cleanup() {
	eb.subscribers = make(map[string][]func(interface{}))
	fmt.Println("事件总线清理完成")
}
