package mqtt

import (
	"sync"
	"time"
)

// Message is a single received MQTT message, ready to ship to the frontend.
// Payload is base64-encoded so binary payloads survive JSON transport.
type Message struct {
	Topic     string `json:"topic"`
	Payload   string `json:"payload"` // base64
	QoS       byte   `json:"qos"`
	Retained  bool   `json:"retained"`
	Timestamp int64  `json:"ts"` // unix millis
}

// batcher buffers incoming messages and flushes them on an interval, so a busy
// broker cannot overwhelm the UI with one event per message.
type batcher struct {
	mu       sync.Mutex
	buf      []Message
	flush    func([]Message)
	interval time.Duration
	done     chan struct{}
	wg       sync.WaitGroup
}

func newBatcher(interval time.Duration, flush func([]Message)) *batcher {
	b := &batcher{
		flush:    flush,
		interval: interval,
		done:     make(chan struct{}),
	}
	b.wg.Add(1)
	go b.loop()
	return b
}

func (b *batcher) add(m Message) {
	b.mu.Lock()
	b.buf = append(b.buf, m)
	b.mu.Unlock()
}

func (b *batcher) loop() {
	defer b.wg.Done()
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.drain()
		case <-b.done:
			b.drain()
			return
		}
	}
}

func (b *batcher) drain() {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = nil
	b.mu.Unlock()
	b.flush(batch)
}

func (b *batcher) stop() {
	close(b.done)
	b.wg.Wait()
}
