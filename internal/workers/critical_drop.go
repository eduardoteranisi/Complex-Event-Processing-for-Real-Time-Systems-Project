package workers

import (
	"cep-module5/internal/domain"
	"github.com/google/uuid"
	"log"
	"time"
)

type OutputQueues interface {
	PushPersistence(event domain.ComplexEvent) bool
	PushNotification(event domain.ComplexEvent) bool
}

type CriticalDropWorker struct {
	h3Map   *H3Map
	outputs OutputQueues
}

func NewCriticalDropWorker(outputs OutputQueues, maxEntities uint64) *CriticalDropWorker {
	return &CriticalDropWorker{
		h3Map:   NewH3Map(maxEntities),
		outputs: outputs,
	}
}

func (w *CriticalDropWorker) Start(inChan <-chan domain.TelemetryEvent) {
	log.Println("[Critical Drop Worker] Thread Curto-Circuito Ativada")

	windowTicker := time.NewTicker(60 * time.Second)
	defer windowTicker.Stop()

	for {
		select {
		case event := <-inChan:
			isRootCause, cluster := w.h3Map.AddEvent(event.Local[0], event.Local[1], event.EventType)

			if isRootCause {
				complexEvent := domain.ComplexEvent{
					CriticalEventID:   uuid.New().String(),
					CriticalEventType: cluster.EventType,
					Local:             [2]float64{cluster.FirstLat, cluster.FirstLon},
					Timestamp:         time.Now().UnixMilli(),
					ClusterSize:       cluster.Count,
				}
				if w.outputs != nil {
					w.outputs.PushPersistence(complexEvent)
					w.outputs.PushNotification(complexEvent)
				}
			}

		case <-windowTicker.C:
			w.h3Map.ResetWindow()
			log.Println("[Critical Drop Worker] Janela de 60s fechada e memória limpa.")
		}
	}
}
