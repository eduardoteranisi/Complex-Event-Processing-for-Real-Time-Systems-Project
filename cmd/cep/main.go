package main

import (
	"log"

	"cep-module5/internal/core"
	"cep-module5/internal/ingest"
	"cep-module5/internal/output"
)

func main() {
	log.Println("[SISTEMA] Iniciando Módulo 5a (Pipeline SEDA Completo: 4 Estágios)...")

	// ==========================================
	// 1. PRÉ-ALOCAÇÃO DE MEMÓRIA (Zero-Allocation)
	// ==========================================

	// Estágio 1: Buffer de Ingestão (Equivalente aos 150 MB)
	ingestBufferSize := uint64(100000)
	ringBuffer := ingest.NewRingBuffer(ingestBufferSize)

	// Estágio 2: Tabela Hash H3 (Equivalente aos 128 MB)
	maxEntities := uint64(350000)
	h3Map := core.NewH3Map(maxEntities)

	// Estágios 3 e 4: Filas de Saída (Equivalente aos 32 MB particionados)
	// Como o struct ComplexEvent é maior, alocamos tamanhos proporcionais aos 24MB e 8MB
	persistenceQueue := output.NewComplexEventRingBuffer(50000)  // Maior tolerância a falhas do DB
	notificationQueue := output.NewComplexEventRingBuffer(15000) // Prioridade de rede, fila menor

	sedaQueues := &output.SEDAQueues{
		Persistence:  persistenceQueue,
		Notification: notificationQueue,
	}

	log.Println("[SISTEMA] Memória estática alocada. Inicializando Threads...")

	// ==========================================
	// 2. INICIALIZAÇÃO DOS 4 ESTÁGIOS (Goroutines)
	// ==========================================

	// [Thread 1] UDP Receiver (Ingestão bruta na porta 9999)
	receiver := ingest.NewUDPReceiver(":9999", ringBuffer)
	go func() {
		if err := receiver.Start(); err != nil {
			log.Fatalf("[SISTEMA] Erro fatal no Receiver UDP: %v", err)
		}
	}()

	// [Thread 3] DB Writer (Consome a fila de persistência)
	dbWriter := output.NewDBWriter(persistenceQueue)
	go dbWriter.Start()

	// [Thread 4] UDP Broadcaster (Consome a fila de notificação e envia para a porta 8888)
	broadcaster := output.NewUDPBroadcaster(notificationQueue, "127.0.0.1:8888")
	go broadcaster.Start()

	// [Thread 2] CEP Core (O Cérebro)
	// Executa na thread principal (bloqueante) e orquestra a passagem de dados
	engine := core.NewCEPEngine(ringBuffer, h3Map, sedaQueues, receiver)
	engine.Start()
}
