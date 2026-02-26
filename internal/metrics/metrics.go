package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Declaramos todas as métricas globais do sistema aqui.
// Note que as variáveis começam com letra Maiúscula para serem exportadas (públicas).

var (
	// Métrica A: Vazão de Ingestão (Se quiser, pode trazer aquele contador do udp_receiver para cá também!)
	EventosRecebidosTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cep_eventos_recebidos_total",
		Help: "Número total de pacotes UDP recebidos na porta de ingestão",
	})

	// Métrica B: Saturação das Filas
	TamanhoFilas = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cep_filas_tamanho_atual",
		Help: "Número de eventos aguardando processamento dentro das filas",
	}, []string{"nome_fila"})

	// Métrica C: Latência de Fechamento (Requisito 2.9.2)
	LatenciaProcessamentoJanela = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "cep_janela_processamento_segundos",
		Help:    "Tempo de processamento pós-fechamento da janela",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 15, 20, 25, 30}, // Foco no limite de 30s
	})
)
