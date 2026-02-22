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

	WriteQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circular_write_queue_depth",
		Help: "Current number of in-memory write requests waiting to be persisted.",
	})

	WriteSpoolDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circular_write_spool_depth",
		Help: "Current number of persistent spool rows waiting to be applied.",
	})

	WriteQueueEnqueuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_enqueued_total",
		Help: "Total number of write requests accepted into the in-memory queue.",
	})

	WriteQueueDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_dropped_total",
		Help: "Total number of write requests dropped from in-memory enqueue due to backpressure.",
	})

	WriteQueueSpilledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_spilled_total",
		Help: "Total number of write requests spooled to persistent storage.",
	})

	WriteQueueRetryTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_retry_total",
		Help: "Total number of persistent spool retries.",
	})

	WriteQueueApplyErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_apply_errors_total",
		Help: "Total number of write batch apply errors.",
	})

	WriteQueueProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circular_write_queue_processed_total",
		Help: "Total number of write requests successfully applied.",
	})

	WriteQueueFlushLatencySeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "circular_write_queue_flush_seconds",
		Help:    "Latency for applying a write batch.",
		Buckets: prometheus.DefBuckets,
	})
)
