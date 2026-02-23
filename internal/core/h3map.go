package core

import (
	"github.com/uber/h3-go/v4"
)

// ClusterState representa o agregador de estado em memória para cada célula H3.
type ClusterState struct {
	H3Index   uint64
	Count     int
	EventType string
	FirstLat  float64
	FirstLon  float64
	Triggered bool // Evita disparar múltiplos alertas para o mesmo hexágono na mesma janela
}

// H3Map implementa uma Tabela Hash Zero-Allocation com Open Addressing.
type H3Map struct {
	nodes       []ClusterState
	usedIndices []uint64 // Guarda os índices usados para limpar o mapa em O(N ativos) ao fim da janela
	capacity    uint64
	activeCount int
}

// NewH3Map pré-aloca a memória (os ~128MB exigidos) na inicialização.
func NewH3Map(maxEntities uint64) *H3Map {
	// Para manter o fator de carga < 0.5 (evitar colisões), dobramos a capacidade máxima
	capacity := maxEntities * 2

	return &H3Map{
		nodes:       make([]ClusterState, capacity), // Alocação estática do Pool de Objetos
		usedIndices: make([]uint64, maxEntities),    // Array auxiliar para reset rápido da janela
		capacity:    capacity,
		activeCount: 0,
	}
}

// AddEvent processa um novo evento e retorna se ele engatilhou a causa raiz.
// Executa em tempo constante O(1).
func (m *H3Map) AddEvent(lat, lon float64, eventType string) (isRootCause bool, state ClusterState) {
	// 1. Biblioteca H3 entra em ação: Converte Lat/Lon para o Hexágono H3 (Resolução 9 é padrão para ruas/bairros)
	latLng := h3.NewLatLng(lat, lon)
	cell, _ := h3.LatLngToCell(latLng, 9)
	h3Index := uint64(cell)

	// 2. Cálculo do Hash e Sondagem Linear (Open Addressing)
	idx := h3Index % m.capacity

	for {
		// Slot vazio: novo cluster encontrado!
		if m.nodes[idx].H3Index == 0 {
			m.nodes[idx].H3Index = h3Index
			m.nodes[idx].Count = 1
			m.nodes[idx].EventType = eventType
			m.nodes[idx].FirstLat = lat
			m.nodes[idx].FirstLon = lon
			m.nodes[idx].Triggered = false

			// Guarda a posição para limpar rápido depois
			m.usedIndices[m.activeCount] = idx
			m.activeCount++

			return false, m.nodes[idx]
		}

		// Slot ocupado pelo nosso hexágono: incrementa contador
		if m.nodes[idx].H3Index == h3Index {
			m.nodes[idx].Count++

			// Regra de Disparo: Clusterização Mínima de 30 Eventos
			if m.nodes[idx].Count >= 30 && !m.nodes[idx].Triggered {
				m.nodes[idx].Triggered = true // Trava para disparar só uma vez
				return true, m.nodes[idx]
			}
			return false, m.nodes[idx]
		}

		// Colisão: avança para o próximo slot (Sondagem Linear)
		idx = (idx + 1) % m.capacity
	}
}

// ResetWindow limpa a tabela para a próxima janela de 60 segundos.
// É executado instantaneamente sem acionar o Garbage Collector.
func (m *H3Map) ResetWindow() {
	for i := 0; i < m.activeCount; i++ {
		slot := m.usedIndices[i]
		m.nodes[slot] = ClusterState{} // Zera o estado reaproveitando a memória
	}
	m.activeCount = 0
}
