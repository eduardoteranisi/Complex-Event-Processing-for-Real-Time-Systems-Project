package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type TelemetryEvent struct {
	EventID   string     `json:"event_id"`
	EventType string     `json:"event_type"`
	Timestamp int64      `json:"timestamp"`
	Local     [2]float64 `json:"local"`
}

var (
	bgRate    int64 = 5000 // Pacotes por segundo
	errorRate int64 = 5    // Porcentagem de pacotes corrompidos
)

func main() {
	log.Println("[Módulo 5b] Iniciando Painel de Controle do Simulador...")

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9999")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	targetZones := [][2]float64{
		{-18.9100, -48.2700}, // Centro
		{-18.9400, -48.2900}, // Sul
		{-18.8800, -48.2500}, // Norte
		{-18.9000, -48.2200}, // Leste
	}

	go startBackgroundNoise(conn, targetZones)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		printMenu()
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		switch input {
		case "1":
			fmt.Print("Digite a nova taxa de pacotes/segundo (ex: 10000): ")
			if scanner.Scan() {
				newRate, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
				if err == nil {
					atomic.StoreInt64(&bgRate, newRate)
					fmt.Printf("✅ Taxa atualizada para %d pacotes/s\n", newRate)
				}
			}
		case "2":
			fmt.Print("Digite a nova taxa de erros (0 a 100%): ")
			if scanner.Scan() {
				newErr, err := strconv.ParseInt(strings.TrimSpace(scanner.Text()), 10, 64)
				if err == nil {
					atomic.StoreInt64(&errorRate, newErr)
					fmt.Printf("✅ Taxa de erro atualizada para %d%%\n", newErr)
				}
			}
		case "3":
			triggerStorm(conn, targetZones, "CRITICAL_DROP", 35)
		case "4":
			triggerStorm(conn, targetZones, "OVERVOLTAGE", 40)
		case "5":
			triggerStorm(conn, targetZones, "UNDERVOLTAGE", 50)

		case "6":
			fmt.Println("\nINICIANDO CAOS TOTAL: Injetando múltiplas anomalias simultâneas...")

			triggerStorm(conn, targetZones, "CRITICAL_DROP", 35)
			triggerStorm(conn, targetZones, "OVERVOLTAGE", 40)
			triggerStorm(conn, targetZones, "UNDERVOLTAGE", 50)

		case "0":
			fmt.Println("Encerrando simulador...")
			os.Exit(0)
		default:
			fmt.Println("❌ Opção inválida.")
		}
	}
}

func triggerStorm(conn *net.UDPConn, zones [][2]float64, eventType string, count int) {
	fmt.Printf("\n⚡ DISPARANDO TEMPESTADE: %s (%d eventos por zona)\n", eventType, count)

	for _, zone := range zones {
		for i := 0; i < count; i++ {
			event := TelemetryEvent{
				EventID:   uuid.New().String(),
				EventType: eventType,
				Timestamp: time.Now().UnixMilli(),
				Local:     [2]float64{zone[0], zone[1]}, // Cravado na zona para clusterizar
			}
			payload, _ := json.Marshal(event)
			conn.Write(payload)
		}
	}
	fmt.Println("✅ Tempestade enviada com sucesso!")
}

func startBackgroundNoise(conn *net.UDPConn, zones [][2]float64) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	basicEvents := []string{"VOLTAGE_DROP", "NOISE", "PING"}

	for range ticker.C {
		currentRate := int(atomic.LoadInt64(&bgRate))
		currentErrRate := int(atomic.LoadInt64(&errorRate))

		for i := 0; i < currentRate; i++ {
			if rand.Intn(100) < currentErrRate {
				conn.Write([]byte(`{"event_id": "corrompido", json_quebrado`))
				continue
			}

			zone := zones[rand.Intn(len(zones))]
			eventType := basicEvents[rand.Intn(len(basicEvents))]

			event := TelemetryEvent{
				EventID:   uuid.New().String(),
				EventType: eventType,
				Timestamp: time.Now().UnixMilli(),
				Local:     [2]float64{zone[0] + (rand.Float64() - 0.5), zone[1] + (rand.Float64() - 0.5)},
			}

			payload, _ := json.Marshal(event)
			conn.Write(payload)
		}
	}
}

func printMenu() {
	fmt.Printf("\n--- PAINEL DO SIMULADOR ---\n")
	fmt.Printf("Carga: %d/s | Erros: %d%%\n", atomic.LoadInt64(&bgRate), atomic.LoadInt64(&errorRate))
	fmt.Println("[ 1 ] - Alterar Carga de Fundo (Pacotes/s)")
	fmt.Println("[ 2 ] - Alterar Taxa de Pacotes Corrompidos (%)")
	fmt.Println("[ 3 ] - ⚡ Disparar Tempestade: CRITICAL_DROP")
	fmt.Println("[ 4 ] - ⚡ Disparar Tempestade: OVERVOLTAGE")
	fmt.Println("[ 5 ] - ⚡ Disparar Tempestade: UNDERVOLTAGE")
	fmt.Println("[ 6 ] - 🌪️  Disparar TODAS as Tempestades Simultaneamente (Apocalipse)")
	fmt.Println("[ 0 ] - Sair")
	fmt.Print("Escolha uma opção: ")
}
