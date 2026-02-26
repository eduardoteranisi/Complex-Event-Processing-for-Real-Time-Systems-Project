package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/google/uuid"
)

type TelemetryEvent struct {
	EventID   string     `json:"event_id"`
	EventType string     `json:"event_type"`
	Timestamp int64      `json:"timestamp"`
	Local     [2]float64 `json:"local"`
}

func main() {
	log.Println("[Módulo 5b] Iniciando Simulador Multi-Regiões...")

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Simulando 4 regiões distintas da cidade afetadas pela mesma tempestade
	targetZones := [][2]float64{
		{-18.9100, -48.2700}, // Centro
		{-18.9400, -48.2900}, // Zona Sul
		{-18.8800, -48.2500}, // Zona Norte
		{-18.9000, -48.2200}, // Zona Leste
	}

	// 1. Carga de Fundo (Background Noise): 5.000 pacotes/s espalhados pela cidade
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for i := 0; i < 5000; i++ {
				if rand.Intn(100) < 5 {
					conn.Write([]byte(`{"event_id": "corrompido", json_quebrado`))
					continue
				}

				// Sorteia uma das 4 zonas para gerar um evento de telemetria normal (sem causar alarme)
				zone := targetZones[rand.Intn(len(targetZones))]

				event := TelemetryEvent{
					EventID:   uuid.New().String(),
					EventType: "VOLTAGE_DROP",
					Timestamp: time.Now().UnixMilli(),
					// Adiciona um ruído (jitter) geográfico amplo para não agrupar no mesmo hexágono facilmente
					Local: [2]float64{zone[0] + (rand.Float64() - 0.5), zone[1] + (rand.Float64() - 0.5)},
				}

				payload, _ := json.Marshal(event)
				conn.Write(payload)
			}
		}
	}()

	// 2. Anomalia Coordenada Multi-Regiões
	// A cada 10 segundos, injeta 35 pacotes cravados em TODAS as 4 zonas ao MESMO TEMPO
	stormTicker := time.NewTicker(10 * time.Second)
	defer stormTicker.Stop()

	for range stormTicker.C {
		log.Println("[Módulo 5b] ⚡ Tempestade severa! Injetando anomalias simultâneas em 4 zonas...")

		for _, zone := range targetZones {
			for i := 0; i < 35; i++ {
				event := TelemetryEvent{
					EventID:   uuid.New().String(),
					EventType: "CRITICAL_DROP",
					Timestamp: time.Now().UnixMilli(),
					Local:     [2]float64{zone[0], zone[1]}, // Coordenada cravada da zona atual
				}
				payload, _ := json.Marshal(event)
				conn.Write(payload)
			}
		}
	}
}
