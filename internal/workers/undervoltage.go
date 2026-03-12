package workers

import (
	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
	"github.com/google/uuid"
	"log"
	"time"
)

type UndervoltageWorker struct {
	h3Map   *H3Map
	outputs OutputQueues
}

func NewUndervoltageWorker(outputs OutputQueues, maxEntities uint64) *UndervoltageWorker {
	return &UndervoltageWorker{
		h3Map:   NewH3Map(maxEntities),
		outputs: outputs,
	}
}

func (w *UndervoltageWorker) Start(inChan <-chan domain.TelemetryEvent) {
	log.Println("[Undervoltage Worker] Thread Undervoltage Ativada")

	windowTicker := time.NewTicker(60 * time.Second)
	defer windowTicker.Stop()

	for {
		select {
		case event := <-inChan:
			startCycle := time.Now()

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

			cycleDuration := time.Since(startCycle).Seconds()

			metrics.LatenciaProcessamentoJanela.WithLabelValues("UNDERVOLTAGE").Observe(cycleDuration)

		case <-windowTicker.C:
			w.h3Map.ResetWindow()
			log.Println("[Undervoltage Worker] Janela de 60s fechada e memória limpa.")
		}
	}
}
