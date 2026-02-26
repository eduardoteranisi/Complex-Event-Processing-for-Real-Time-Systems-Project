package output

import (
	"sync/atomic"

	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
)

// ComplexEventRingBuffer é a versão Lock-Free para as filas de saída.
type ComplexEventRingBuffer struct {
	data                   []domain.ComplexEvent
	capacity               uint64
	head                   uint64
	tail                   uint64
	complexEventBufferName string
}

func NewComplexEventRingBuffer(name string, capacity uint64) *ComplexEventRingBuffer {
	return &ComplexEventRingBuffer{
		data:                   make([]domain.ComplexEvent, capacity),
		capacity:               capacity,
		complexEventBufferName: name,
	}
}

func (rb *ComplexEventRingBuffer) Push(event domain.ComplexEvent) bool {
	t := atomic.LoadUint64(&rb.tail)
	h := atomic.LoadUint64(&rb.head)
	if h-t >= rb.capacity {
		return false
	}
	rb.data[h%rb.capacity] = event
	atomic.AddUint64(&rb.head, 1)

	metrics.TamanhoFilas.WithLabelValues(rb.complexEventBufferName).Inc()
	return true
}

func (rb *ComplexEventRingBuffer) Pop() (domain.ComplexEvent, bool) {
	h := atomic.LoadUint64(&rb.head)
	t := atomic.LoadUint64(&rb.tail)
	if t == h {
		return domain.ComplexEvent{}, false
	}
	event := rb.data[t%rb.capacity]
	atomic.AddUint64(&rb.tail, 1)

	metrics.TamanhoFilas.WithLabelValues(rb.complexEventBufferName).Dec()
	return event, true
}

// SEDAQueues agrupa as duas filas de saída implementando a interface do CEP Core.
type SEDAQueues struct {
	Persistence  *ComplexEventRingBuffer
	Notification *ComplexEventRingBuffer
}

// PushPersistence atende à interface OutputQueues
func (q *SEDAQueues) PushPersistence(event domain.ComplexEvent) bool {
	return q.Persistence.Push(event)
}

// PushNotification atende à interface OutputQueues
func (q *SEDAQueues) PushNotification(event domain.ComplexEvent) bool {
	return q.Notification.Push(event)
}
