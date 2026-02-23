package output

import (
	"encoding/json"
	"log"
	"net"
	"time"
)

type UDPBroadcaster struct {
	queue   *ComplexEventRingBuffer
	address string
}

func NewUDPBroadcaster(queue *ComplexEventRingBuffer, address string) *UDPBroadcaster {
	return &UDPBroadcaster{
		queue:   queue,
		address: address, // Ex: "255.255.255.255:8888" ou "127.0.0.1:8888" (para testes locais)
	}
}

func (b *UDPBroadcaster) Start() {
	log.Printf("[Broadcaster] Iniciando Thread de Notificação UDP para %s...", b.address)

	addr, err := net.ResolveUDPAddr("udp", b.address)
	if err != nil {
		log.Fatalf("[Broadcaster] Erro ao resolver endereço: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("[Broadcaster] Erro ao abrir conexão UDP: %v", err)
	}
	defer conn.Close()

	for {
		event, ok := b.queue.Pop()
		if !ok {
			time.Sleep(1 * time.Millisecond) // Fila vazia, descansa a CPU
			continue
		}

		// Serializa o Incidente de Causa Raiz
		payload, err := json.Marshal(event)
		if err != nil {
			log.Printf("[Broadcaster] Erro ao serializar JSON: %v", err)
			continue
		}

		// Dispara na rede (UDP)
		_, err = conn.Write(payload)
		if err != nil {
			log.Printf("[Broadcaster] Falha ao enviar alerta de causa raiz: %v", err)
		} else {
			log.Printf("[Broadcaster] Alerta enviado via UDP para o Módulo 3! ID: %s", event.CriticalEventID)
		}
	}
}
