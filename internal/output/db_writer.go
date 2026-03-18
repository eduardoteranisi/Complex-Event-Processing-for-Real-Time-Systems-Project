package output

import (
	"database/sql"
	"log"
	"time"

	"cep-module5/internal/domain"
	"cep-module5/internal/metrics"
	_ "github.com/lib/pq"
)

type DBWriter struct {
	persistenceChan chan domain.ComplexEvent
	dlqChan         chan domain.ComplexEvent
	db              *sql.DB
}

func NewDBWriter(persistenceChan chan domain.ComplexEvent, db *sql.DB) *DBWriter {
	return &DBWriter{
		persistenceChan: persistenceChan,
		db:              db,
		dlqChan:         make(chan domain.ComplexEvent, 10000),
	}
}

func (dbWriter *DBWriter) Start() {
	log.Println("[DB Writer] Iniciando Thread de Persistência Assíncrona...")
	defer dbWriter.db.Close()

	insertSQL := `
        INSERT INTO incident_log (critical_event_id, critical_event_type, latitude, longitude, event_timestamp, cluster_size)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	stmt, err := dbWriter.db.Prepare(insertSQL)
	if err != nil {
		log.Fatalf("[DB Writer] Erro ao preparar statement SQL: %v", err)
	}
	defer stmt.Close()

	for event := range dbWriter.persistenceChan {

		metrics.TamanhoFilas.WithLabelValues("persistence").Dec()

		success := false

		// Mecanismo de Retry para Falhas Transientes do Banco
		for i := 0; i < 3; i++ {
			_, err := stmt.Exec(
				event.CriticalEventID,
				event.CriticalEventType,
				event.Local[0],
				event.Local[1],
				event.Timestamp,
				event.ClusterSize,
			)

			if err == nil {
				log.Printf("[DB Writer] 💾 Incidente gravado no PostgreSQL. ID: %s", event.CriticalEventID)
				success = true
				break
			}

			log.Printf("[DB Writer] ⚠️ Banco fora do ar ou erro de Insert. Retentando em 2 segundos... Erro: %v", err)
			time.Sleep(2 * time.Second)
		}

		// DEAD LETTER QUEUE
		if !success {
			log.Printf("🚨 ERRO CRÍTICO: Banco inacessível após 3 tentativas. Salvando em DLQ local.")
			dbWriter.saveToDLQ(event)
		}
	}
}

func (dbWriter *DBWriter) saveToDLQ(event domain.ComplexEvent) {
	select {
	case dbWriter.dlqChan <- event:
		metrics.TamanhoFilas.WithLabelValues("dlq").Inc()

		log.Println("⚠️ [DB Writer] Evento enviado para a DLQ em memória para reprocessamento futuro.")
	default:
		log.Println("❌ FALHA CATASTRÓFICA: Fila da DLQ cheia! Evento descartado definitivamente.")
	}
}

func (dbWriter *DBWriter) StartDLQConsumer() {
	log.Println("[DLQ Consumer] Iniciando trabalhador de recuperação de eventos...")

	insertSQL := `
        INSERT INTO incident_log (critical_event_id, critical_event_type, latitude, longitude, event_timestamp, cluster_size)
        VALUES ($1, $2, $3, $4, $5, $6)
    `

	go func() {
		for event := range dbWriter.dlqChan {
			metrics.TamanhoFilas.WithLabelValues("dlq").Dec()

			time.Sleep(5 * time.Second)

			_, err := dbWriter.db.Exec(insertSQL,
				event.CriticalEventID,
				event.CriticalEventType,
				event.Local[0],
				event.Local[1],
				event.Timestamp,
				event.ClusterSize,
			)

			if err != nil {
				log.Printf("❌ [DLQ Consumer] Falha ao recuperar evento. Reenfileirando... Erro: %v", err)
				dbWriter.saveToDLQ(event) // Volta para o final da fila em memória
			} else {
				log.Println("✅ [DLQ Consumer] Evento recuperado da DLQ e salvo no banco com sucesso!")
			}
		}
	}()
}
