package workers

import (
	"github.com/uber/h3-go/v4"
)

// Define a resolucao (precisao) da biblioteca H3
const H3Resolution int = 9
const GenericWorkerH3Resolution int = 7

type ClusterState struct {
	H3Index   uint64
	Count     int
	EventType string
	FirstLat  float64
	FirstLon  float64
	Triggered bool
}

type H3Map struct {
	nodes       []ClusterState
	usedIndices []uint64
	capacity    uint64
	activeCount int
}

func NewH3Map(maxEntities uint64) *H3Map {
	capacity := maxEntities * 2

	return &H3Map{
		nodes:       make([]ClusterState, capacity),
		usedIndices: make([]uint64, maxEntities),
		capacity:    capacity,
		activeCount: 0,
	}
}

func (m *H3Map) AddEvent(lat, lon float64, eventType string) (isRootCause bool, state ClusterState) {
	latLng := h3.NewLatLng(lat, lon)
	cell, _ := h3.LatLngToCell(latLng, H3Resolution)
	h3Index := uint64(cell)

	idx := h3Index % m.capacity

	for {
		if m.nodes[idx].H3Index == 0 {
			m.nodes[idx].H3Index = h3Index
			m.nodes[idx].Count = 1
			m.nodes[idx].EventType = eventType
			m.nodes[idx].FirstLat = lat
			m.nodes[idx].FirstLon = lon
			m.nodes[idx].Triggered = false

			m.usedIndices[m.activeCount] = idx
			m.activeCount++

			return false, m.nodes[idx]
		}

		if m.nodes[idx].H3Index == h3Index {
			m.nodes[idx].Count++

			if m.nodes[idx].Count >= 30 && !m.nodes[idx].Triggered {
				m.nodes[idx].Triggered = true
				return true, m.nodes[idx]
			}
			return false, m.nodes[idx]
		}

		idx = (idx + 1) % m.capacity
	}
}

func (m *H3Map) ResetWindow() {
	for i := 0; i < m.activeCount; i++ {
		slot := m.usedIndices[i]
		m.nodes[slot] = ClusterState{}
	}
	m.activeCount = 0
}
