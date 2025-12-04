package opensearch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus 指标变量
var (
	// 批量操作指标
	bulkRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "requests_total",
			Help:      "Total number of bulk requests",
		},
		[]string{"status", "index"},
	)

	bulkDocumentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "documents_total",
			Help:      "Total number of documents in bulk operations",
		},
		[]string{"status", "index"},
	)

	bulkBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "bytes_total",
			Help:      "Total bytes sent in bulk operations",
		},
		[]string{"index"},
	)

	bulkDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "duration_seconds",
			Help:      "Duration of bulk operations in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"index"},
	)

	bulkRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "retries_total",
			Help:      "Total number of bulk operation retries",
		},
		[]string{"index"},
	)

	bulkQueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "bulk",
			Name:      "queue_size",
			Help:      "Current size of the bulk indexer queue",
		},
		[]string{"index"},
	)

	// 查询操作指标
	queryDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "opensearch",
			Subsystem: "query",
			Name:      "duration_seconds",
			Help:      "Duration of query operations in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		},
		[]string{"index", "query_type"},
	)

	queryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "query",
			Name:      "total",
			Help:      "Total number of query operations",
		},
		[]string{"status", "index"},
	)

	queryHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "query",
			Name:      "hits_total",
			Help:      "Total number of hits returned by queries",
		},
		[]string{"index"},
	)

	// 连接池指标
	connectionPoolSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "connection",
			Name:      "pool_size",
			Help:      "Current connection pool size",
		},
		[]string{"host"},
	)

	connectionPoolActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "connection",
			Name:      "pool_active",
			Help:      "Number of active connections in the pool",
		},
		[]string{"host"},
	)

	// 集群健康指标
	clusterHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "cluster",
			Name:      "health",
			Help:      "Cluster health status (0=red, 1=yellow, 2=green)",
		},
		[]string{"cluster"},
	)

	clusterNodesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "cluster",
			Name:      "nodes_total",
			Help:      "Total number of nodes in the cluster",
		},
		[]string{"cluster"},
	)

	clusterShardsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "cluster",
			Name:      "shards_active",
			Help:      "Number of active shards",
		},
		[]string{"cluster"},
	)

	clusterShardsUnassigned = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "opensearch",
			Subsystem: "cluster",
			Name:      "shards_unassigned",
			Help:      "Number of unassigned shards",
		},
		[]string{"cluster"},
	)

	// 索引操作指标
	indexOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Subsystem: "index",
			Name:      "operations_total",
			Help:      "Total number of index operations",
		},
		[]string{"operation", "status"},
	)

	// 错误指标
	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "opensearch",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"type", "operation"},
	)
)

// Metrics 提供指标记录的辅助方法
type Metrics struct {
	enabled bool
}

// NewMetrics 创建指标记录器
func NewMetrics(enabled bool) *Metrics {
	return &Metrics{enabled: enabled}
}

// RecordBulkRequest 记录批量请求
func (m *Metrics) RecordBulkRequest(index string, status string, docs int, bytes int64, duration float64) {
	if !m.enabled {
		return
	}
	bulkRequestsTotal.WithLabelValues(status, index).Inc()
	bulkDocumentsTotal.WithLabelValues(status, index).Add(float64(docs))
	bulkBytesTotal.WithLabelValues(index).Add(float64(bytes))
	bulkDurationSeconds.WithLabelValues(index).Observe(duration)
}

// RecordBulkRetry 记录批量重试
func (m *Metrics) RecordBulkRetry(index string) {
	if !m.enabled {
		return
	}
	bulkRetriesTotal.WithLabelValues(index).Inc()
}

// RecordBulkQueueSize 记录批量队列大小
func (m *Metrics) RecordBulkQueueSize(index string, size int) {
	if !m.enabled {
		return
	}
	bulkQueueSize.WithLabelValues(index).Set(float64(size))
}

// RecordQuery 记录查询操作
func (m *Metrics) RecordQuery(index string, queryType string, status string, hits int64, duration float64) {
	if !m.enabled {
		return
	}
	queryTotal.WithLabelValues(status, index).Inc()
	queryDurationSeconds.WithLabelValues(index, queryType).Observe(duration)
	if hits > 0 {
		queryHitsTotal.WithLabelValues(index).Add(float64(hits))
	}
}

// RecordClusterHealth 记录集群健康状态
func (m *Metrics) RecordClusterHealth(cluster string, health *ClusterHealth) {
	if !m.enabled || health == nil {
		return
	}

	// 转换状态为数值
	var statusValue float64
	switch health.Status {
	case "green":
		statusValue = 2
	case "yellow":
		statusValue = 1
	case "red":
		statusValue = 0
	}

	clusterHealth.WithLabelValues(cluster).Set(statusValue)
	clusterNodesTotal.WithLabelValues(cluster).Set(float64(health.NumberOfNodes))
	clusterShardsActive.WithLabelValues(cluster).Set(float64(health.ActiveShards))
	clusterShardsUnassigned.WithLabelValues(cluster).Set(float64(health.UnassignedShards))
}

// RecordIndexOperation 记录索引操作
func (m *Metrics) RecordIndexOperation(operation string, status string) {
	if !m.enabled {
		return
	}
	indexOperationsTotal.WithLabelValues(operation, status).Inc()
}

// RecordError 记录错误
func (m *Metrics) RecordError(errorType string, operation string) {
	if !m.enabled {
		return
	}
	errorsTotal.WithLabelValues(errorType, operation).Inc()
}

// RecordConnectionPool 记录连接池状态
func (m *Metrics) RecordConnectionPool(host string, size int, active int) {
	if !m.enabled {
		return
	}
	connectionPoolSize.WithLabelValues(host).Set(float64(size))
	connectionPoolActive.WithLabelValues(host).Set(float64(active))
}

// defaultMetrics 默认指标记录器
var defaultMetrics = NewMetrics(true)

// SetDefaultMetrics 设置默认指标记录器
func SetDefaultMetrics(m *Metrics) {
	defaultMetrics = m
}

// GetDefaultMetrics 获取默认指标记录器
func GetDefaultMetrics() *Metrics {
	return defaultMetrics
}
