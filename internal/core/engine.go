package core

import (
	"log"
	"time"

	"github.com/google/uuid"

	"cep-module5/internal/domain"
	"cep-module5/internal/ingest"
)

// OutputQueues define o contrato para as filas de saída (Estágios 3 e 4).
// A implementação real garantirá o Zero-Allocation.
type OutputQueues interface {
	PushPersistence(event domain.ComplexEvent) bool
	PushNotification(event domain.ComplexEvent) bool
}

// CEPEngine é o orquestrador do Estágio 2.
type CEPEngine struct {
	inputBuffer *ingest.RingBuffer
	h3Map       *H3Map
	outputs     OutputQueues
}

// NewCEPEngine constrói o motor de processamento ligando a entrada, a memória de estado e a saída.
func NewCEPEngine(input *ingest.RingBuffer, h3Map *H3Map, outputs OutputQueues) *CEPEngine {
	return &CEPEngine{
		inputBuffer: input,
		h3Map:       h3Map,
		outputs:     outputs,
	}
}

// Start inicia o loop principal da Thread Core.
// Deve ser executado em uma única Goroutine para garantir o determinismo SPSC.
func (e *CEPEngine) Start() {
	log.Println("[CEP Core] Iniciando motor de correlação temporal/espacial...")

	// O sistema opera sob um modelo de janelas de tempo discretas de 60 segundos
	windowTicker := time.NewTicker(60 * time.Second)
	defer windowTicker.Stop()

	for {
		select {
		case <-windowTicker.C:
			// Fechamento da Janela: Reseta a tabela hash em O(N ativos) sem alocar memória
			log.Println("[CEP Core] Janela de 60s fechada. Resetando agregadores de estado...")
			e.h3Map.ResetWindow()

		default:
			// 1. Tenta retirar um evento do Ring Buffer (Wait-Free)
			event, ok := e.inputBuffer.Pop()

			if !ok {
				// Buffer vazio: dorme 1 milissegundo para evitar 100% de uso de CPU em idle
				time.Sleep(1 * time.Millisecond)
				continue
			}

			// 2. Processa a agregação espacial (H3) em O(1)
			isRootCause, cluster := e.h3Map.AddEvent(event.Local[0], event.Local[1], event.EventType)

			// 3. Verifica a regra de negócio (Disparo de Alarme)
			if isRootCause {
				// Cria o Incidente de Causa Raiz consolidado (Tabela 3.5.2)
				complexEvent := domain.ComplexEvent{
					CriticalEventID:   uuid.New().String(),
					CriticalEventType: cluster.EventType,
					Local:             [2]float64{cluster.FirstLat, cluster.FirstLon},
					Timestamp:         time.Now().UnixMilli(),
					ClusterSize:       cluster.Count,
				}

				log.Printf("Agrupamento anormal de %d quedas de energia na coordenada [%.4f, %.4f] neste minuto",
					complexEvent.ClusterSize, complexEvent.Local[0], complexEvent.Local[1])

				// 4. Despacha simultaneamente para as filas independentes
				if e.outputs != nil {
					e.outputs.PushPersistence(complexEvent)
					e.outputs.PushNotification(complexEvent)
				}
			}
		}
	}
}
