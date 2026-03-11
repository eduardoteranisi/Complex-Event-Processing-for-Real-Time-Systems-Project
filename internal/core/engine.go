package core

import (
	"log"
	"time"

	"cep-module5/internal/domain"
	"cep-module5/internal/ingest"
	"cep-module5/internal/workers"
)

type OutputQueues interface {
	PushPersistence(event domain.ComplexEvent) bool
	PushNotification(event domain.ComplexEvent) bool
}

type CEPEngine struct {
	inputBuffer *ingest.RingBuffer
	outputs     OutputQueues
	receiver    *ingest.UDPReceiver
	maxEntities uint64
}

func NewCEPEngine(input *ingest.RingBuffer, outputs OutputQueues, receiver *ingest.UDPReceiver, maxEntities uint64) *CEPEngine {
	return &CEPEngine{
		inputBuffer: input,
		outputs:     outputs,
		receiver:    receiver,
		maxEntities: maxEntities,
	}
}

func (e *CEPEngine) Start() {
	log.Println("[CEP Router] Iniciando o Roteador de Eventos (Demultiplexer)...")

	// ==========================================
	// 1. INICIALIZAÇÃO DAS ESTEIRAS E WORKERS
	// ==========================================

	genericChan := make(chan domain.TelemetryEvent, 1000)
	criticalDropChan := make(chan domain.TelemetryEvent, 1000)
	overvoltageChan := make(chan domain.TelemetryEvent, 1000)
	undervoltageChan := make(chan domain.TelemetryEvent, 1000)

	genericWorker := workers.NewGenericWorker(e.outputs, 5)
	go genericWorker.Start(genericChan)

	criticalDropWorker := workers.NewCriticalDropWorker(e.outputs, e.maxEntities)
	go criticalDropWorker.Start(criticalDropChan)

	overvoltageWorker := workers.NewUndervoltageWorker(e.outputs, 5)
	go overvoltageWorker.Start(overvoltageChan)

	undervoltageWorker := workers.NewUndervoltageWorker(e.outputs, e.maxEntities)
	go undervoltageWorker.Start(undervoltageChan)

	// ==========================================
	// 2. CONFIGURAÇÃO DA OBSERVABILIDADE DE INGESTÃO
	// ==========================================

	reportDuration := 60 * time.Second
	reportTicker := time.NewTicker(reportDuration)
	defer reportTicker.Stop()

	var lastRecv, lastDrop uint64
	windowStartTime := time.Now()

	// ==========================================
	// 3. LOOP PRINCIPAL (ALTA VAZÃO)
	// ==========================================
	for {
		select {
		case <-reportTicker.C:
			windowProcessTime := time.Since(windowStartTime)
			currentRecv, currentDrop := e.receiver.GetMetrics()

			recvThisWindow := currentRecv - lastRecv
			dropThisWindow := currentDrop - lastDrop

			log.Printf("\n=== RELATÓRIO SINTÉTICO DE INGESTÃO (60s) ===")
			log.Printf("Pacotes Recebidos   : %d", recvThisWindow)
			log.Printf("Pacotes Descartados : %d", dropThisWindow)
			log.Printf("Uptime da Janela    : %v", windowProcessTime)
			log.Printf("=============================================\n")

			lastRecv = currentRecv
			lastDrop = currentDrop
			windowStartTime = time.Now()

		default:
			event, ok := e.inputBuffer.Pop()
			if !ok {
				time.Sleep(1 * time.Millisecond)
				continue
			}

			// Caso de saturacao do buffer circular: motor esta super lento e demora 10 milissegundos para processar
			//time.Sleep(10 * time.Millisecond)

			switch event.EventType {

			case "CRITICAL_DROP":
				criticalDropChan <- event

			case "OVERVOLTAGE":
				overvoltageChan <- event

			case "UNDERVOLTAGE":
				undervoltageChan <- event

				//default:
				//	genericChan <- event
			}
		}
	}
}
