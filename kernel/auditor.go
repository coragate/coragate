package kernel

import (
	"context"
	"log"
	"sync"
)

// Auditor 内存队列 + 后台 flush。Enqueue 不得等待适配器落盘。
type Auditor struct {
	store Store
	ch    chan Envelope
	done  chan struct{}
	once  sync.Once
}

// NewAuditor 启动后台 worker。store 为 nil 时 Enqueue 为空操作。
func NewAuditor(store Store, buf int) *Auditor {
	if store == nil {
		return nil
	}
	if buf < 1 {
		buf = 64
	}
	a := &Auditor{
		store: store,
		ch:    make(chan Envelope, buf),
		done:  make(chan struct{}),
	}
	go a.loop()
	return a
}

func (a *Auditor) loop() {
	defer close(a.done)
	for env := range a.ch {
		if err := a.store.Append(context.Background(), env); err != nil {
			log.Printf("audit flush: %v", err)
		}
	}
}

// Enqueue 非阻塞交给队列；队列满则丢弃（进程崩溃同样会丢未 flush 缓冲）。
func (a *Auditor) Enqueue(env Envelope) {
	if a == nil {
		return
	}
	select {
	case a.ch <- env:
	default:
	}
}

// Close 等队列排空后停止 worker。
func (a *Auditor) Close() {
	if a == nil {
		return
	}
	a.once.Do(func() { close(a.ch) })
	<-a.done
}

// List 只读查询已 flush 的 envelope。控制面看板走本接口，不直接 SELECT 存储实现。
func (a *Auditor) List(ctx context.Context, limit int) ([]Envelope, error) {
	if a == nil || a.store == nil {
		return nil, nil
	}
	return a.store.List(ctx, limit)
}
