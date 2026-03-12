package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Métrica A: Vazão de Ingestão
	EventosRecebidosTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cep_eventos_recebidos_total",
		Help: "Número total de pacotes UDP recebidos na porta de ingestão",
	})

	// Métrica B: Saturação das Filas
	TamanhoFilas = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cep_filas_tamanho_atual",
		Help: "Número de eventos aguardando processamento dentro das filas",
	}, []string{"nome_fila"})

	// Métrica C: Latência de Processamento
	LatenciaProcessamentoJanela = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cep_janela_processamento_segundos",
		Help:    "Tempo de processamento pós-fechamento da janela",
		Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.1, 1, 30},
	}, []string{"worker"})
)
