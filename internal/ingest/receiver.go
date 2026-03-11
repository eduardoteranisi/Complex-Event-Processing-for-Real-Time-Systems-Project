package ingest

import (
	"encoding/json"
	"log"
	"net"
	"sync/atomic"

	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
)

type UDPReceiver struct {
	address     string
	ringBuffer  *RingBuffer
	packetsRecv uint64 // Contador atómico para observabilidade
	packetsDrop uint64 // Contador atómico de erros (Silent Drop)
}

func NewUDPReceiver(address string, rb *RingBuffer) *UDPReceiver {
	return &UDPReceiver{
		address:    address,
		ringBuffer: rb,
	}
}

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

	readBuffer := make([]byte, 1024)

	var event domain.TelemetryEvent

	for {
		n, _, err := conn.ReadFromUDP(readBuffer)
		if err != nil {
			atomic.AddUint64(&r.packetsDrop, 1)
			continue // Silent Drop: Ignora o erro de rede e avança
		}

		atomic.AddUint64(&r.packetsRecv, 1)

		metrics.EventosRecebidosTotal.Inc()

		err = json.Unmarshal(readBuffer[:n], &event)
		if err != nil {
			atomic.AddUint64(&r.packetsDrop, 1)
			continue // Silent Drop: Pacote malformado, tipo incorreto ou corrompido
		}

		success := r.ringBuffer.Push(event)
		if !success {
			atomic.AddUint64(&r.packetsDrop, 1) // Silent Drop: Buffer cheio (Overrun)
		}
	}
}

func (r *UDPReceiver) GetMetrics() (received uint64, dropped uint64) {
	return atomic.LoadUint64(&r.packetsRecv), atomic.LoadUint64(&r.packetsDrop)
}
