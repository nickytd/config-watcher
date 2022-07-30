package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "cfgw"
)

var (
	calcHashes = promauto.NewCounter(prometheus.CounterOpts{
		Name: prometheus.BuildFQName(namespace, "hash", "calculated_total"),
		Help: "Total number of calculated hashes.",
	})

	totalHashesUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: prometheus.BuildFQName(namespace, "hash", "updates_total"),
		Help: "Total number updated hashes",
	})

	fileHashes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: prometheus.BuildFQName(namespace, "file", "hash"),
		Help: "Repesents calculated file hash in a watched directory",
	},
		[]string{"file", "hash", "total_hash"},
	)

	processRestarts = promauto.NewCounter(prometheus.CounterOpts{
		Name: prometheus.BuildFQName(namespace, "process", "restarts_total"),
		Help: "Total number of processes restarts.",
	})
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

func ProcssesRestarts() {
	processRestarts.Inc()
}
