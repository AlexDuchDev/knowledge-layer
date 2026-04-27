// Package metrics owns the Prometheus registry exposed by the API's /metrics
// endpoint and provides typed recorders that domain packages call after work
// completes (job runs, connector syncs, …) plus on-scrape collectors that
// surface live infra state (Postgres pool, Asynq queue depth) without
// requiring background goroutines.
//
// Why a dedicated package: domain modules like knowledge_jobs cannot import
// internal/httpserver, but both need to share one registry so that metrics
// recorded inside a job processor show up at /metrics. This package is the
// single source of truth; httpserver, knowledge_jobs, and ingestion_connectors
// all depend on it (one direction, no cycles).
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	once sync.Once

	registry *prometheus.Registry

	httpRequests           *prometheus.CounterVec
	jobRunDuration         *prometheus.HistogramVec
	connectorSyncDuration  *prometheus.HistogramVec
)

// init lazily wires the registry, base collectors (Go, process), and the
// fixed CounterVec/HistogramVec shapes. Custom on-scrape collectors (pool
// stats, queue depth) are registered by callers via RegisterPoolStats /
// RegisterQueueDepth.
func ensure() {
	once.Do(func() {
		registry = prometheus.NewRegistry()

		httpRequests = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests by method and Fiber route template (low cardinality).",
			},
			[]string{"method", "route"},
		)
		jobRunDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "knowledge_job_run_duration_seconds",
				Help:    "Knowledge job run duration in seconds, partitioned by job_type and final status.",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s .. ~409s
			},
			[]string{"job_type", "status"},
		)
		connectorSyncDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "connector_sync_duration_seconds",
				Help:    "Connector source feed sync duration in seconds, partitioned by adapter_kind and status.",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 12),
			},
			[]string{"adapter_kind", "status"},
		)

		registry.MustRegister(httpRequests)
		registry.MustRegister(jobRunDuration)
		registry.MustRegister(connectorSyncDuration)
		registry.MustRegister(collectors.NewGoCollector())
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	})
}

// Gatherer returns the shared Prometheus gatherer. Mounted by /metrics in httpserver.
func Gatherer() prometheus.Gatherer {
	ensure()
	return registry
}

// HTTPRequestsCounter is exposed for the Fiber middleware that increments per-request counts.
func HTTPRequestsCounter() *prometheus.CounterVec {
	ensure()
	return httpRequests
}

// ObserveJobRun records the duration and final status of a knowledge job run.
// Status is a short string ("completed" | "failed"); avoid free-form values.
func ObserveJobRun(jobType, status string, dur time.Duration) {
	ensure()
	jobRunDuration.WithLabelValues(jobType, status).Observe(dur.Seconds())
}

// ObserveConnectorSync records the duration and final status of a sync run.
func ObserveConnectorSync(adapterKind, status string, dur time.Duration) {
	ensure()
	connectorSyncDuration.WithLabelValues(adapterKind, status).Observe(dur.Seconds())
}

// RegisterPoolStats wires an on-scrape collector that reports pgxpool stats
// (total connections, idle, in-use, acquire counts/durations). Idempotent: a
// second call with the same pool no-ops the registration.
func RegisterPoolStats(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	ensure()
	c := newPoolCollector(pool)
	// Re-registration of an identical collector returns AlreadyRegisteredError
	// which we ignore so callers can be naive.
	if err := registry.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// Best effort: log via panic-recovery would be wrong here; metrics
			// are non-critical. Silent failure is acceptable per Prometheus
			// guidance ("metrics never break the program").
			_ = err
		}
	}
}

// RegisterQueueDepth wires an on-scrape collector that reports Asynq queue
// stats (pending, active, scheduled, retry, archived, failed) per queue.
// Idempotent in the same way as RegisterPoolStats.
func RegisterQueueDepth(inspector *asynq.Inspector, queues []string) {
	if inspector == nil || len(queues) == 0 {
		return
	}
	ensure()
	c := newQueueDepthCollector(inspector, queues)
	if err := registry.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			_ = err
		}
	}
}

// --- Custom collectors --------------------------------------------------

type poolCollector struct {
	pool *pgxpool.Pool

	totalDesc      *prometheus.Desc
	idleDesc       *prometheus.Desc
	acquiredDesc   *prometheus.Desc
	maxDesc        *prometheus.Desc
	acquireCount   *prometheus.Desc
	acquireSecs    *prometheus.Desc
}

func newPoolCollector(pool *pgxpool.Pool) *poolCollector {
	return &poolCollector{
		pool:         pool,
		totalDesc:    prometheus.NewDesc("postgres_pool_total_conns", "Total connections currently in the pgx pool.", nil, nil),
		idleDesc:     prometheus.NewDesc("postgres_pool_idle_conns", "Connections currently idle.", nil, nil),
		acquiredDesc: prometheus.NewDesc("postgres_pool_acquired_conns", "Connections currently checked out.", nil, nil),
		maxDesc:      prometheus.NewDesc("postgres_pool_max_conns", "Configured pool max size.", nil, nil),
		acquireCount: prometheus.NewDesc("postgres_pool_acquire_count_total", "Cumulative successful acquires.", nil, nil),
		acquireSecs:  prometheus.NewDesc("postgres_pool_acquire_duration_seconds_total", "Cumulative time spent acquiring connections.", nil, nil),
	}
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalDesc
	ch <- c.idleDesc
	ch <- c.acquiredDesc
	ch <- c.maxDesc
	ch <- c.acquireCount
	ch <- c.acquireSecs
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	st := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, float64(st.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue, float64(st.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.acquiredDesc, prometheus.GaugeValue, float64(st.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.maxDesc, prometheus.GaugeValue, float64(st.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(st.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireSecs, prometheus.CounterValue, st.AcquireDuration().Seconds())
}

type queueDepthCollector struct {
	inspector *asynq.Inspector
	queues    []string

	pendingDesc   *prometheus.Desc
	activeDesc    *prometheus.Desc
	scheduledDesc *prometheus.Desc
	retryDesc     *prometheus.Desc
	archivedDesc  *prometheus.Desc
	failedDesc    *prometheus.Desc
}

func newQueueDepthCollector(insp *asynq.Inspector, queues []string) *queueDepthCollector {
	return &queueDepthCollector{
		inspector:     insp,
		queues:        queues,
		pendingDesc:   prometheus.NewDesc("asynq_queue_pending", "Pending tasks per Asynq queue.", []string{"queue"}, nil),
		activeDesc:    prometheus.NewDesc("asynq_queue_active", "Active tasks per Asynq queue.", []string{"queue"}, nil),
		scheduledDesc: prometheus.NewDesc("asynq_queue_scheduled", "Scheduled (delayed) tasks per Asynq queue.", []string{"queue"}, nil),
		retryDesc:     prometheus.NewDesc("asynq_queue_retry", "Tasks awaiting retry per Asynq queue.", []string{"queue"}, nil),
		archivedDesc:  prometheus.NewDesc("asynq_queue_archived", "Archived (terminally failed) tasks per Asynq queue.", []string{"queue"}, nil),
		failedDesc:    prometheus.NewDesc("asynq_queue_failed", "Tasks failed in the most recent stats window per Asynq queue.", []string{"queue"}, nil),
	}
}

func (c *queueDepthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.pendingDesc
	ch <- c.activeDesc
	ch <- c.scheduledDesc
	ch <- c.retryDesc
	ch <- c.archivedDesc
	ch <- c.failedDesc
}

func (c *queueDepthCollector) Collect(ch chan<- prometheus.Metric) {
	if c.inspector == nil {
		return
	}
	// Bound Inspector calls so a slow Redis cannot stall a /metrics scrape.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = ctx // asynq.Inspector methods are blocking; the timeout is advisory until
	// the upstream API accepts a context. We still keep the variable for future
	// migration without churn.
	for _, q := range c.queues {
		info, err := c.inspector.GetQueueInfo(q)
		if err != nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.pendingDesc, prometheus.GaugeValue, float64(info.Pending), q)
		ch <- prometheus.MustNewConstMetric(c.activeDesc, prometheus.GaugeValue, float64(info.Active), q)
		ch <- prometheus.MustNewConstMetric(c.scheduledDesc, prometheus.GaugeValue, float64(info.Scheduled), q)
		ch <- prometheus.MustNewConstMetric(c.retryDesc, prometheus.GaugeValue, float64(info.Retry), q)
		ch <- prometheus.MustNewConstMetric(c.archivedDesc, prometheus.GaugeValue, float64(info.Archived), q)
		ch <- prometheus.MustNewConstMetric(c.failedDesc, prometheus.GaugeValue, float64(info.Failed), q)
	}
}
