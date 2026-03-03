package workers

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/uber/h3-go/v4"

	"cep-module5/internal/domain"
)

type GenericWorker struct {
	outputs    OutputQueues
	muteMap    map[string]time.Time
	mutePeriod time.Duration
}

func NewGenericWorker(outputs OutputQueues, muteMinutes int) *GenericWorker {
	return &GenericWorker{
		outputs:    outputs,
		muteMap:    make(map[string]time.Time),
		mutePeriod: time.Duration(muteMinutes) * time.Minute,
	}
}

func (w *GenericWorker) Start(inChan <-chan domain.TelemetryEvent) {
	log.Printf("[Worker Genérico] Proteção Anti-Flood ativada. Silenciamento: %v", w.mutePeriod)

	cleanupTicker := time.NewTicker(60 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case event := <-inChan:
			latLng := h3.NewLatLng(event.Local[0], event.Local[1])
			cell, err := h3.LatLngToCell(latLng, H3Resolution)

			var h3Index string
			if err != nil {
				h3Index = fmt.Sprintf("%.2f_%.2f", event.Local[0], event.Local[1])
				log.Printf("[Worker Genérico] Aviso: Falha no H3 para [%.4f, %.4f]. Usando fallback.", event.Local[0], event.Local[1])
			} else {
				h3Index = cell.String()
			}

			cacheKey := event.EventType + "_" + h3Index

			lastTime, isMuted := w.muteMap[cacheKey]

			if !isMuted || time.Since(lastTime) > w.mutePeriod {

				basicAlarm := domain.ComplexEvent{
					CriticalEventID:   uuid.New().String(),
					CriticalEventType: "ALARME_BASICO: " + event.EventType,
					Local:             event.Local,
					Timestamp:         time.Now().UnixMilli(),
					ClusterSize:       1,
				}

				if w.outputs != nil {
					w.outputs.PushNotification(basicAlarm)
					// w.outputs.PushPersistence(basicAlarm)
				}

				w.muteMap[cacheKey] = time.Now()
				log.Printf("[Router Básico] ⚠️ Alerta '%s' despachado. Silenciando zona por %v.", event.EventType, w.mutePeriod)
			}

		case <-cleanupTicker.C:
			agora := time.Now()
			for key, lastTime := range w.muteMap {
				if agora.Sub(lastTime) > w.mutePeriod {
					delete(w.muteMap, key)
				}
			}
		}
	}
}
