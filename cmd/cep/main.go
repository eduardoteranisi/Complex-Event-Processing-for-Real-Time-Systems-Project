package main

import (
	"log"

	"cep-module5/internal/core"
	"cep-module5/internal/ingest"
)

func main() {
	log.Println("[SISTEMA] Iniciando Módulo 5a (Teste de Estágios 1 e 2)...")

	// ==========================================
	// 1. PRÉ-ALOCAÇÃO (A regra de ouro)
	// ==========================================
	bufferSize := uint64(100000) // Buffer de Ingestão
	ringBuffer := ingest.NewRingBuffer(bufferSize)

	maxEntities := uint64(350000) // Capacidade para 350k células H3
	h3Map := core.NewH3Map(maxEntities)

	log.Println("[SISTEMA] Memória pré-alocada com sucesso (Zero-Allocation garantido).")

	// ==========================================
	// 2. INICIALIZAÇÃO DOS ESTÁGIOS
	// ==========================================

	// Estágio 1: Receiver UDP (Thread 1)
	receiver := ingest.NewUDPReceiver(":9999", ringBuffer)
	go func() {
		if err := receiver.Start(); err != nil {
			log.Fatalf("Erro fatal no Receiver UDP: %v", err)
		}
	}()

	// Estágio 2: Motor CEP (Thread 2)
	// Passamos 'nil' no terceiro parâmetro porque ainda não temos as filas de saída
	engine := core.NewCEPEngine(ringBuffer, h3Map, nil)

	// O Start() do motor possui um loop infinito, então ele trava a Goroutine principal aqui,
	// mantendo o programa vivo e processando os eventos.
	engine.Start()
}
