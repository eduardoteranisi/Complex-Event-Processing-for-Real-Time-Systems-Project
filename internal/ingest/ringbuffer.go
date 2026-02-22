package ingest

import (
	"sync/atomic"

	"cep-module5a/internal/domain"
)

// RingBuffer implementa uma fila circular pre-alocada e lock-free para o padrão SPSC.
type RingBuffer struct {
	data     []domain.TelemetryEvent
	capacity uint64
	head     uint64 // Ponteiro de escrita (Produtor / UDP Receiver)
	tail     uint64 // Ponteiro de leitura (Consumidor / CEP Core)
}

// NewRingBuffer realiza a alocação estática na inicialização (Heap Pre-allocation).
// ATENÇÃO: Esta função só pode ser chamada UMA VEZ no main.go.
func NewRingBuffer(capacity uint64) *RingBuffer {
	return &RingBuffer{
		data:     make([]domain.TelemetryEvent, capacity), // Alocação contígua e estática
		capacity: capacity,
		head:     0,
		tail:     0,
	}
}

// Push insere um novo evento no buffer. 
// Operação Wait-Free executada pela Thread de Ingestão (UDP).
func (rb *RingBuffer) Push(event domain.TelemetryEvent) bool {
	// Lê os ponteiros atomicamente direto do cache da CPU
	t := atomic.LoadUint64(&rb.tail)
	h := atomic.LoadUint64(&rb.head)

	// Verifica se o buffer está cheio (head alcançou a tail + capacidade)
	// Com a folga de 150MB, isso só aconteceria se o CEP Core travasse completamente.
	if h-t >= rb.capacity {
		return false // Buffer overrun (Fila cheia)
	}

	// Insere o dado no índice circular
	rb.data[h%rb.capacity] = event
	
	// Incrementa a head atomicamente (Memory Barrier - libera a leitura para o Core)
	atomic.AddUint64(&rb.head, 1)
	return true
}

// Pop retira um evento do buffer para processamento.
// Operação Wait-Free executada pela Thread do Motor CEP.
func (rb *RingBuffer) Pop() (domain.TelemetryEvent, bool) {
	h := atomic.LoadUint64(&rb.head)
	t := atomic.LoadUint64(&rb.tail)

	// Verifica se o buffer está vazio
	if t == h {
		return domain.TelemetryEvent{}, false
	}

	// Resgata o dado
	event := rb.data[t%rb.capacity]
	
	// Incrementa a tail atomicamente (libera o espaço para o UDP Receiver)
	atomic.AddUint64(&rb.tail, 1)
	return event, true
}
