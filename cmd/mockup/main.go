package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/google/uuid"
)

// TelemetryEvent espelha a entrada do Módulo 5a
type TelemetryEvent struct {
	EventID   string     `json:"event_id"`
	EventType string     `json:"event_type"`
	Timestamp int64      `json:"timestamp"`
	Local     [2]float64 `json:"local"`
}

func main() {
	log.Println("[Módulo 5b] Iniciando Simulador de Tempestade de Alarmes...")

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Coordenadas alvo para forçar o agrupamento (Causa Raiz)
	targetLat, targetLon := -18.9100, -48.2700

	// Dispara a carga em rajadas de 1 segundo (5.000 pacotes/segundo = 300.000/minuto)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Executa 5.000 envios de rede o mais rápido que a CPU aguentar neste segundo
			for i := 0; i < 5000; i++ {
				// 5% de chance de enviar um pacote corrompido para testar o Silent Drop
				if rand.Intn(100) < 5 {
					conn.Write([]byte(`{"event_id": "corrompido", json_quebrado`))
					continue
				}

				// Gera pacote normal espalhado aleatoriamente em torno da coordenada alvo
				event := TelemetryEvent{
					EventID:   uuid.New().String(),
					EventType: "VOLTAGE_DROP",
					Timestamp: time.Now().UnixMilli(),
					Local:     [2]float64{targetLat + (rand.Float64() - 0.5), targetLon + (rand.Float64() - 0.5)},
				}

				payload, _ := json.Marshal(event)
				conn.Write(payload) // Chamada de sistema (Syscall) UDP
			}
		}
	}()

	// Rotina maliciosa: A cada 10 segundos, injeta 35 pacotes simultâneos na MESMA coordenada
	// Isso garante que o CEP Core (Módulo 5a) dispare o alerta de Causa Raiz
	stormTicker := time.NewTicker(10 * time.Second)
	defer stormTicker.Stop()

	for range stormTicker.C {
		log.Println("[Módulo 5b] ⚡ Injetando Anomalia Coordenada (35 pacotes no mesmo hexágono)...")
		for i := 0; i < 35; i++ {
			event := TelemetryEvent{
				EventID:   uuid.New().String(),
				EventType: "CRITICAL_DROP",
				Timestamp: time.Now().UnixMilli(),
				Local:     [2]float64{targetLat, targetLon}, // Mesma coordenada exata!
			}
			payload, _ := json.Marshal(event)
			conn.Write(payload)
		}
	}
}
