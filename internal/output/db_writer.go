package output

import (
	"log"
	"time"
)

type DBWriter struct {
	queue *ComplexEventRingBuffer
}

func NewDBWriter(queue *ComplexEventRingBuffer) *DBWriter {
	return &DBWriter{
		queue: queue,
	}
}

func (db *DBWriter) Start() {
	log.Println("[DB Writer] Iniciando Thread de Persistência no Incident Log...")

	for {
		event, ok := db.queue.Pop()
		if !ok {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// SIMULAÇÃO DE LENTIDÃO DE DISCO/REDE DO BANCO DE DADOS (Ex: 200ms)
		// Como estamos desacoplados, isso não vai afetar a Thread 2 (CEP Core) nem a Thread 4 (Broadcaster)!
		time.Sleep(200 * time.Millisecond)

		log.Printf("[DB Writer] 💾 Incidente gravado no SGBD com sucesso. ID: %s | Cluster: %d falhas",
			event.CriticalEventID, event.ClusterSize)
	}
}
