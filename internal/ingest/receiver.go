package ingest

import (
	"encoding/json"
	"log"
	"net"
	"sync/atomic"

	"cep-module5/internal/domain"
)

// UDPReceiver é responsável por escutar a rede e alimentar o RingBuffer.
type UDPReceiver struct {
	address     string
	ringBuffer  *RingBuffer
	packetsRecv uint64 // Contador atómico para observabilidade
	packetsDrop uint64 // Contador atómico de erros (Silent Drop)
}

// NewUDPReceiver cria uma nova instância do recetor UDP.
func NewUDPReceiver(address string, rb *RingBuffer) *UDPReceiver {
	return &UDPReceiver{
		address:    address,
		ringBuffer: rb,
	}
}

// Start inicia o loop infinito de escuta e processamento. Deve ser executado numa Goroutine.
func (r *UDPReceiver) Start() error {
	addr, err := net.ResolveUDPAddr("udp", r.address)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("[Receiver] A escutar UDP Broadcast em %s", r.address)

	// =========================================================================
	// ESTRATÉGIA ZERO-ALLOCATION (HOT PATH)
	// O buffer de leitura é pré-alocado FORA do loop.
	// 1024 bytes é mais do que suficiente para o payload JSON de 250 bytes.
	// =========================================================================
	readBuffer := make([]byte, 1024)

	// Variável de evento reutilizada para evitar alocações dinâmicas
	var event domain.TelemetryEvent

	// Loop infinito de altíssima prioridade (Estágio 1 do Pipeline SEDA)
	for {
		// 1. Leitura bloqueante e super rápida diretamente para o buffer estático
		n, _, err := conn.ReadFromUDP(readBuffer)
		if err != nil {
			atomic.AddUint64(&r.packetsDrop, 1)
			continue // Silent Drop: Ignora o erro de rede e avança
		}

		atomic.AddUint64(&r.packetsRecv, 1)

		// 2. Desserialização (Parsing do JSON)
		err = json.Unmarshal(readBuffer[:n], &event)
		if err != nil {
			atomic.AddUint64(&r.packetsDrop, 1)
			continue // Silent Drop: Pacote malformado, tipo incorreto ou corrompido
		}

		// 3. Inserção no Ring Buffer (Wait-Free)
		success := r.ringBuffer.Push(event)
		if !success {
			atomic.AddUint64(&r.packetsDrop, 1) // Silent Drop: Buffer cheio (Overrun)
		}
	}
}

// GetMetrics devolve as métricas atuais para o relatório de observabilidade (Requisito 2.9.4)
func (r *UDPReceiver) GetMetrics() (received uint64, dropped uint64) {
	return atomic.LoadUint64(&r.packetsRecv), atomic.LoadUint64(&r.packetsDrop)
}
