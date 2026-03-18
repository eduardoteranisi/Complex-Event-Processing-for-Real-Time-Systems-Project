package output

import (
	"encoding/json"
	"log"
	"net"

	"cep-module5/internal/domain"
	"cep-module5/internal/metrics" // Adicionado para manter o painel de filas preciso
)

type UDPBroadcaster struct {
	// Substituímos o RingBuffer pelo canal nativo
	notificationChan chan domain.ComplexEvent
	address          string
}

// O construtor agora recebe o canal diretamente
func NewUDPBroadcaster(notificationChan chan domain.ComplexEvent, address string) *UDPBroadcaster {
	return &UDPBroadcaster{
		notificationChan: notificationChan,
		address:          address,
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

	// O 'range' consome a fila nativamente sem desperdiçar CPU
	for event := range b.notificationChan {

		// Decrementa a métrica do Grafana assim que o evento sai da fila
		metrics.TamanhoFilas.WithLabelValues("notification").Dec()

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
