package ingest

import (
	"sync/atomic"

	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
)

type RingBuffer struct {
	data      []domain.TelemetryEvent
	capacity  uint64
	head      uint64
	tail      uint64
	queueName string
}

func NewRingBuffer(name string, capacity uint64) *RingBuffer {
	return &RingBuffer{
		data:      make([]domain.TelemetryEvent, capacity),
		capacity:  capacity,
		head:      0,
		tail:      0,
		queueName: name,
	}
}

func (rb *RingBuffer) Push(event domain.TelemetryEvent) bool {
	t := atomic.LoadUint64(&rb.tail)
	h := atomic.LoadUint64(&rb.head)

	if h-t >= rb.capacity {
		return false
	}

	rb.data[h%rb.capacity] = event

	atomic.AddUint64(&rb.head, 1)

	metrics.TamanhoFilas.WithLabelValues(rb.queueName).Inc()
	return true
}

func (rb *RingBuffer) Pop() (domain.TelemetryEvent, bool) {
	h := atomic.LoadUint64(&rb.head)
	t := atomic.LoadUint64(&rb.tail)

	if t == h {
		return domain.TelemetryEvent{}, false
	}

	event := rb.data[t%rb.capacity]

	atomic.AddUint64(&rb.tail, 1)

	metrics.TamanhoFilas.WithLabelValues(rb.queueName).Dec()
	return event, true
}
