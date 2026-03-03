package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cep-module5/internal/core"
	"cep-module5/internal/ingest"
	"cep-module5/internal/output"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: Arquivo .env não encontrado. Tentando ler do sistema...")
	}

	udpTarget := os.Getenv("MOD3_UDP_TARGET")

	if udpTarget == "" {
		log.Println("⚠️ Aviso: MOD3_UDP_TARGET vazio. Usando 127.0.0.1:8888 por padrão.")
		udpTarget = "127.0.0.1:8888"
	} else {
		log.Printf("📡 Alvo UDP configurado para: %s\n", udpTarget)
	}

	log.Println("[SISTEMA] Iniciando Módulo 5a (Pipeline SEDA Completo: 4 Estágios)...")

	// ==========================================
	// 0. SERVIDOR DE MÉTRICAS (PROMETHEUS)
	// ==========================================
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("[Monitoramento] 📊 Prometheus escutando na porta :2112 (/metrics)")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Erro no servidor de métricas: %v", err)
		}
	}()
	// ==========================================
	// 1. PRÉ-ALOCAÇÃO DE MEMÓRIA (Zero-Allocation)
	// ==========================================

	// Estágio 1: Buffer de Ingestão (Equivalente aos 150 MB)
	ingestBufferSize := uint64(100000)
	ringBuffer := ingest.NewRingBuffer("ingestion", ingestBufferSize)

	// Estágio 2: Tabela Hash H3 (Equivalente aos 128 MB)
	maxEntities := uint64(350000)

	// Estágios 3 e 4: Filas de Saída (Equivalente aos 32 MB particionados)
	persistenceQueue := output.NewComplexEventRingBuffer("persistence", 50000)
	notificationQueue := output.NewComplexEventRingBuffer("notification", 15000)

	sedaQueues := &output.SEDAQueues{
		Persistence:  persistenceQueue,
		Notification: notificationQueue,
	}

	log.Println("[SISTEMA] Memória estática alocada. Fazendo conexao com o banco Incident Log...")

	// ==========================================
	// 1.5. CONEXÃO COM O BANCO E RECUPERAÇÃO DE ESTADO
	// ==========================================
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("postgres", connStr)
	if err != nil || db.Ping() != nil {
		log.Fatalf("[SISTEMA] ❌ Erro fatal: Banco de dados inatingível.")
	}
	defer db.Close()
	log.Println("[SISTEMA] ✅ Conectado ao Incident Log. Inicializando threads...")

	recovery := output.NewDBRecovery(db)

	umaHoraAtras := time.Now().Add(-1 * time.Hour).UnixMilli()
	historico := recovery.RecoverRecentIncidents(umaHoraAtras)

	for _, eventoPassado := range historico {
		notificationQueue.Push(eventoPassado)
	}
	// ==========================================
	// 2. INICIALIZAÇÃO DOS 4 ESTÁGIOS (Goroutines)
	// ==========================================

	// [Thread 1] UDP Receiver
	receiver := ingest.NewUDPReceiver(":9999", ringBuffer)
	go func() {
		if err := receiver.Start(); err != nil {
			log.Fatalf("[SISTEMA] Erro fatal no Receiver UDP: %v", err)
		}
	}()

	// [Thread 3] DB Writer
	dbWriter := output.NewDBWriter(persistenceQueue, db)
	go dbWriter.Start()

	// [Thread 4] UDP Broadcaster
	broadcaster := output.NewUDPBroadcaster(notificationQueue, udpTarget)
	go broadcaster.Start()

	// CEP Core (O Cérebro)
	engine := core.NewCEPEngine(ringBuffer, sedaQueues, receiver, maxEntities)
	engine.Start()
}
