package output

import (
	"database/sql"
	"log"

	"cep-module5/internal/domain"
)

type DBRecovery struct {
	db *sql.DB
}

func NewDBRecovery(db *sql.DB) *DBRecovery {
	return &DBRecovery{db: db}
}

// RecoverRecentIncidents busca incidentes a partir de um timestamp específico
func (r *DBRecovery) RecoverRecentIncidents(sinceUnixMilli int64) []domain.ComplexEvent {
	log.Println("[Recovery] Iniciando varredura no Incident Log para restaurar o estado...")

	query := `
		SELECT critical_event_id, critical_event_type, latitude, longitude, event_timestamp, cluster_size
		FROM incident_log
		WHERE event_timestamp >= $1
		ORDER BY event_timestamp ASC
	`

	rows, err := r.db.Query(query, sinceUnixMilli)
	if err != nil {
		log.Printf("[Recovery] ❌ Erro ao buscar dados históricos: %v", err)
		return nil
	}
	defer rows.Close()

	var recovered []domain.ComplexEvent

	for rows.Next() {
		var event domain.ComplexEvent
		var lat, lon float64

		err := rows.Scan(
			&event.CriticalEventID,
			&event.CriticalEventType,
			&lat,
			&lon,
			&event.Timestamp,
			&event.ClusterSize,
		)
		if err != nil {
			log.Printf("[Recovery] Erro ao ler linha do backup: %v", err)
			continue
		}

		event.Local = [2]float64{lat, lon}
		recovered = append(recovered, event)
	}

	log.Printf("[Recovery] ✅ Recuperação concluída. %d incidentes restaurados do banco.", len(recovered))
	return recovered
}
