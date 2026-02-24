package output

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DBWriter struct {
	queue *ComplexEventRingBuffer
	db    *sql.DB
}

// NewDBWriter agora inicializa a conexão com o RDBMS.
func NewDBWriter(queue *ComplexEventRingBuffer, db *sql.DB) *DBWriter {
	return &DBWriter{
		queue: queue,
		db:    db,
	}
}

func (dbWriter *DBWriter) Start() {
	log.Println("[DB Writer] Iniciando Thread de Persistência Assíncrona...")
	defer dbWriter.db.Close()

	// Preparamos o statement de Insert uma única vez para máxima performance
	insertSQL := `
		INSERT INTO incident_log (critical_event_id, critical_event_type, latitude, longitude, event_timestamp, cluster_size)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	stmt, err := dbWriter.db.Prepare(insertSQL)
	if err != nil {
		log.Fatalf("[DB Writer] Erro ao preparar statement SQL: %v", err)
	}
	defer stmt.Close()

	for {
		event, ok := dbWriter.queue.Pop()
		if !ok {
			time.Sleep(1 * time.Millisecond) // Fila vazia
			continue
		}

		// NOVO: Mecanismo de Retry Infinito para Falhas Transientes do Banco
		for {
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
				break // Sai do loop de retry e vai buscar o próximo evento na fila
			}

			// Se chegou aqui, o banco recusou a conexão ou deu erro.
			log.Printf("[DB Writer] ⚠️ Banco fora do ar ou erro de Insert. Retentando em 2 segundos... Erro: %v", err)

			// Pausa de 2 segundos (Backoff) para não causar um ataque DDoS no próprio banco
			time.Sleep(2 * time.Second)

			// Nota: Enquanto estamos travados neste Sleep, a arquitetura SEDA brilha.
			// O Motor CEP não sabe que o banco caiu. Ele continua rodando a 15.000 req/s
			// e jogando dados na nossa Fila de Persistência (Ring Buffer) de forma assíncrona.
		}
	}
}
