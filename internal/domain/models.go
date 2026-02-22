package domain

// TelemetryEvent mapeia a Tabela 3.5.1: Esquema de dados de entrada esperado.
// Este é o pacote bruto que chega via UDP Broadcast do Módulo 5b.
type TelemetryEvent struct {
	EventID   string     `json:"event_id"`
	EventType string     `json:"event_type"`
	Timestamp int64      `json:"timestamp"` // Long (Unix Epoch)
	Local     [2]float64 `json:"local"`     // Array fixo [Lat, Lon] para evitar alocação dinâmica (Zero-Allocation)
}

// ComplexEvent mapeia a Tabela 3.5.2: Esquema de dados de saída esperado.
// Este é o Incidente de Causa Raiz consolidado enviado para o Módulo 3 e Incident Log.
type ComplexEvent struct {
	CriticalEventID   string     `json:"critical_event_id"`
	CriticalEventType string     `json:"critical_event_type"`
	Local             [2]float64 `json:"local"`
	Timestamp         int64      `json:"timestamp"`
	ClusterSize       int        `json:"cluster_size"` // Número de eventos básicos associados
}
