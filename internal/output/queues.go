package output

import (
	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
)

// SEDAQueues agora gerencia canais nativos.
type SEDAQueues struct {
	Persistence  chan domain.ComplexEvent
	Notification chan domain.ComplexEvent
}

// NewSEDAQueues substitui a criação dos Ring Buffers.
func NewSEDAQueues(persistCap, notifyCap int) *SEDAQueues {
	return &SEDAQueues{
		Persistence:  make(chan domain.ComplexEvent, persistCap),
		Notification: make(chan domain.ComplexEvent, notifyCap),
	}
}

func (q *SEDAQueues) PushPersistence(event domain.ComplexEvent) bool {
	select {
	case q.Persistence <- event:
		// Incrementa a métrica que tínhamos no Ring Buffer
		metrics.TamanhoFilas.WithLabelValues("persistence").Inc()
		return true
	default:
		// Fila lotada! Retorna false para o worker saber que houve descarte
		return false
	}
}

func (q *SEDAQueues) PushNotification(event domain.ComplexEvent) bool {
	select {
	case q.Notification <- event:
		metrics.TamanhoFilas.WithLabelValues("notification").Inc()
		return true
	default:
		return false
	}
}
