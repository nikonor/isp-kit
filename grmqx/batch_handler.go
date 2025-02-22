package grmqx

import (
	"context"
	"sync"
	"time"

	"github.com/integration-system/grmq/consumer"
)

type BatchHandlerAdapter interface {
	Handle(batch []BatchItem)
}

type BatchHandlerAdapterFunc func(batch []BatchItem)

func (b BatchHandlerAdapterFunc) Handle(batch []BatchItem) {
	b(batch)
}

type BatchItem struct {
	Context  context.Context
	Delivery *consumer.Delivery
}

type BatchHandler struct {
	adapter       BatchHandlerAdapter
	purgeInterval time.Duration
	maxSize       int
	batch         []BatchItem
	c             chan BatchItem
	runner        *sync.Once
	closed        bool
	lock          sync.Locker
}

func NewBatchHandler(
	adapter BatchHandlerAdapter,
	purgeInterval time.Duration,
	maxSize int,
) *BatchHandler {
	return &BatchHandler{
		adapter:       adapter,
		purgeInterval: purgeInterval,
		maxSize:       maxSize,
		c:             make(chan BatchItem),
		runner:        &sync.Once{},
		lock:          &sync.Mutex{},
	}
}

func (r *BatchHandler) Handle(ctx context.Context, delivery *consumer.Delivery) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		_ = delivery.Nack(true)
	}

	r.runner.Do(func() {
		go r.run()
	})
	r.c <- BatchItem{
		Context:  ctx,
		Delivery: delivery,
	}
}

func (r *BatchHandler) run() {
	defer func() {
		if len(r.batch) > 0 {
			r.adapter.Handle(r.batch)
		}
	}()
	for {
		select {
		case item, ok := <-r.c:
			if !ok {
				return
			}
			r.batch = append(r.batch, item)
			if len(r.batch) < r.maxSize {
				continue
			}
		case <-time.After(r.purgeInterval):
			if len(r.batch) == 0 {
				continue
			}
		}

		r.adapter.Handle(r.batch)
		r.batch = nil
	}
}

func (r *BatchHandler) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.closed = true
	close(r.c)
}
