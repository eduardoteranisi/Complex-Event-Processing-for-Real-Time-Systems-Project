package core

import (
	"log"
	"time"

	"github.com/google/uuid"

	"cep-module5/internal/domain"
	"cep-module5/internal/ingest"
)

// OutputQueues define o contrato para as filas de saída.
type OutputQueues interface {
	PushPersistence(event domain.ComplexEvent) bool
	PushNotification(event domain.ComplexEvent) bool
}

// CEPEngine é o orquestrador do Estágio 2.
type CEPEngine struct {
	inputBuffer *ingest.RingBuffer
	h3Map       *H3Map
	outputs     OutputQueues
	receiver    *ingest.UDPReceiver // Injetado para coletar as métricas de Observabilidade
}

// NewCEPEngine constrói o motor recebendo todas as dependências SEDA.
func NewCEPEngine(input *ingest.RingBuffer, h3Map *H3Map, outputs OutputQueues, receiver *ingest.UDPReceiver) *CEPEngine {
	return &CEPEngine{
		inputBuffer: input,
		h3Map:       h3Map,
		outputs:     outputs,
		receiver:    receiver,
	}
}

// Start inicia o loop principal da Thread Core.
func (e *CEPEngine) Start() {
	log.Println("[CEP Core] Iniciando motor de correlação temporal/espacial...")

	windowDuration := 60 * time.Second
	windowTicker := time.NewTicker(windowDuration)
	defer windowTicker.Stop()

	// Variáveis de controle de métricas da janela atual
	var incidentsThisWindow int
	var lastRecv, lastDrop uint64
	windowStartTime := time.Now()

	for {
		select {
		case <-windowTicker.C:
			// 1. Coleta e cálculo das métricas
			windowProcessTime := time.Since(windowStartTime)
			currentRecv, currentDrop := e.receiver.GetMetrics()

			recvThisWindow := currentRecv - lastRecv
			dropThisWindow := currentDrop - lastDrop

			// 2. Emissão do Relatório de Observabilidade (Requisito 2.9.4)
			log.Printf("\n=== RELATÓRIO SINTÉTICO DA JANELA (60s) ===")
			log.Printf("Pacotes Recebidos        : %d", recvThisWindow)
			log.Printf("Pacotes Descartados      : %d", dropThisWindow)
			log.Printf("Incidentes Identificados : %d", incidentsThisWindow)
			log.Printf("Tempo Total da Janela    : %v", windowProcessTime)
			log.Printf("===========================================\n")

			// 3. Reset de estado para a próxima janela em O(N ativos)
			e.h3Map.ResetWindow()

			// 4. Prepara os contadores para o próximo ciclo
			lastRecv = currentRecv
			lastDrop = currentDrop
			incidentsThisWindow = 0
			windowStartTime = time.Now()

		default:
			event, ok := e.inputBuffer.Pop()
			if !ok {
				time.Sleep(1 * time.Millisecond)
				continue
			}

			isRootCause, cluster := e.h3Map.AddEvent(event.Local[0], event.Local[1], event.EventType)

			if isRootCause {
				incidentsThisWindow++ // Incrementa a métrica do relatório

				complexEvent := domain.ComplexEvent{
					CriticalEventID:   uuid.New().String(),
					CriticalEventType: cluster.EventType,
					Local:             [2]float64{cluster.FirstLat, cluster.FirstLon},
					Timestamp:         time.Now().UnixMilli(),
					ClusterSize:       cluster.Count,
				}

				// Log customizado legível para o operador no console local
				log.Printf("Agrupamento anormal de %d quedas de energia na coordenada [%.4f, %.4f] neste minuto",
					complexEvent.ClusterSize, complexEvent.Local[0], complexEvent.Local[1])

				if e.outputs != nil {
					e.outputs.PushPersistence(complexEvent)
					e.outputs.PushNotification(complexEvent)
				}
			}
		}
	}
}

