package main

import (
	"log"
	"time"

	"cep-module5/internal/ingest"
)

func main() {
	log.Println("Iniciando Módulo 5a (Teste de Ingestão)...")

	// 1. Pré-alocação do Ring Buffer (150 MB / ~600.000 pacotes de margem)
	// Para o teste, vamos alocar um tamanho menor só para ver rodando rápido, ex: 100.000
	bufferSize := uint64(100000)
	ringBuffer := ingest.NewRingBuffer(bufferSize)
	log.Printf("RingBuffer pré-alocado para %d eventos.", bufferSize)

	// 2. Instanciar e iniciar o UDP Receiver na porta 9999
	receiver := ingest.NewUDPReceiver(":9999", ringBuffer)

	// Rodamos o receiver em uma Goroutine separada (Thread de I/O)
	go func() {
		if err := receiver.Start(); err != nil {
			log.Fatalf("Erro fatal no Receiver UDP: %v", err)
		}
	}()

	// 3. Consumidor Fantasma (Simulando o Estágio 2 / CEP Core)
	// Ele vai ficar rodando e a cada 5 segundos imprime as métricas.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var eventosProcessados uint64

	for {
		select {
		case <-ticker.C:
			recv, drop := receiver.GetMetrics()
			log.Printf("[MÉTRICAS] Recebidos: %d | Descartados: %d | Retirados do Buffer: %d", recv, drop, eventosProcessados)
		default:
			// Tenta tirar um evento do buffer
			evento, ok := ringBuffer.Pop()
			if ok {
				eventosProcessados++
				// Imprime o ID do evento só para vermos que chegou íntegro
				log.Printf("[DADO NOVO] Evento válido recebido: %s", evento.EventID)
			} else {
				// Se o buffer estiver vazio, dorme 1 milissegundo para não fritar a CPU no teste
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}
