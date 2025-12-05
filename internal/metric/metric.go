package metric

import "github.com/prometheus/client_golang/prometheus"

var (
	// api metrics
	// request count
	RequestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	// request latency
	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"method", "path", "status"})

	//request error rate
	RequestErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Total number of HTTP request errors.",
	}, []string{"method", "path", "status"})

	// cache metrics
	// cache miss/hit rate
	CacheHits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of cache hits.",
	}, []string{"cache_name"})

	CacheMisses = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of cache misses.",
	}, []string{"cache_name"})

	// key count
	CacheKeyCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_key_count",
		Help: "Number of keys in the cache.",
	}, []string{"cache_name"})

	// expiration count
	CacheExpirations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_expirations_total",
		Help: "Total number of expired keys in the cache.",
	}, []string{"cache_name"})

	// persistence metrics
	// persistence(aof) write count
	AofWriteCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "persistence_aof_write_total",
		Help: "Total number of AOF writes.",
	}, []string{"status"})

	// persistence(aof) write errors
	AofWriteErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "persistence_aof_write_errors_total",
		Help: "Total number of AOF write errors.",
	}, []string{"status"})

	// persistence(aof) write duration
	AofWriteDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "persistence_aof_write_duration_seconds",
		Help: "Duration of AOF writes.",
	}, []string{"status"})

	// persistence(snapshot) write count
	SnapshotWriteCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "persistence_snapshot_write_total",
		Help: "Total number of snapshot writes.",
	}, []string{"status"})

	// persistence(snapshot) write errors
	SnapshotWriteErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "persistence_snapshot_write_errors_total",
		Help: "Total number of snapshot write errors.",
	}, []string{"status"})

	// persistence(snapshot) write duration
	SnapshotWriteDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "persistence_snapshot_write_duration_seconds",
		Help: "Duration of snapshot writes.",
	}, []string{"status"})
)

func init() {
	// register metrics
	// api metrics
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(RequestErrors)
	// cache metrics
	prometheus.MustRegister(CacheHits)
	prometheus.MustRegister(CacheMisses)
	prometheus.MustRegister(CacheKeyCount)
	prometheus.MustRegister(CacheExpirations)
	// persistence metrics
	// AOF
	prometheus.MustRegister(AofWriteCount)
	prometheus.MustRegister(AofWriteErrors)
	prometheus.MustRegister(AofWriteDuration)
	// Snapshot
	prometheus.MustRegister(SnapshotWriteCount)
	prometheus.MustRegister(SnapshotWriteErrors)
	prometheus.MustRegister(SnapshotWriteDuration)
}
