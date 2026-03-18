package output

import (
	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
)

type SEDAQueues struct {
	Persistence  chan domain.ComplexEvent
	Notification chan domain.ComplexEvent
}

func NewSEDAQueues(persistCap, notifyCap int) *SEDAQueues {
	return &SEDAQueues{
		Persistence:  make(chan domain.ComplexEvent, persistCap),
		Notification: make(chan domain.ComplexEvent, notifyCap),
	}
}

func (q *SEDAQueues) PushPersistence(event domain.ComplexEvent) bool {
	select {
	case q.Persistence <- event:
		metrics.TamanhoFilas.WithLabelValues("persistence").Inc()
		return true
	default:
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
