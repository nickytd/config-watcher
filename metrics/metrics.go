package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	calcHashes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cfgw_calculated_hashes_total",
		Help: "The total number of calculated hashes.",
	})

	totalHashesUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cfgw_total_hash_updates_total",
		Help: "The number of total hash updates.",
	})

	fileHashes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cfgw_calculated_files_hashes",
		Help: "Repesents calculated hashes of files in a watched directory",
	},
		[]string{"file", "hash", "total_hash"},
	)
)

func IncreaseCalculatedHashes() {
	calcHashes.Inc()
}

func IncreaseTotalHashUpdates() {
	totalHashesUpdates.Inc()
}

func AddFileHash(file, hash, total_hash string) {
	fileHashes.WithLabelValues(file, hash, total_hash).Set(1)
}

func ResetFileHash() {
	fileHashes.Reset()
}
