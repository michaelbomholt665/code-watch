package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics definitions
var (
	ParsingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "circular_parsing_seconds",
		Help:    "Time spent parsing a source file.",
		Buckets: prometheus.DefBuckets,
	}, []string{"language"})

	GraphNodes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circular_graph_nodes_total",
		Help: "Total number of nodes in the dependency graph.",
	})

	GraphEdges = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circular_graph_edges_total",
		Help: "Total number of edges in the dependency graph.",
	})

	AnalysisDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "circular_analysis_seconds",
		Help:    "Time spent on high-level analysis tasks.",
		Buckets: prometheus.DefBuckets,
	}, []string{"task"})

	WatcherEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_watcher_events_total",
		Help: "Total number of file system events received by the watcher.",
	})
)
