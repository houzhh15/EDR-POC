package database

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var (
	// 连接池指标
	dbPoolOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_open_connections",
		Help: "Number of open connections to the database",
	})

	dbPoolInUseConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_in_use_connections",
		Help: "Number of connections currently in use",
	})

	dbPoolIdleConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_pool_idle_connections",
		Help: "Number of idle connections",
	})

	dbPoolWaitCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "db_pool_wait_total",
		Help: "Total number of connections waited for",
	})

	dbPoolWaitDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_pool_wait_duration_seconds",
		Help:    "Time blocked waiting for a new connection",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	})

	// 查询指标
	dbQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"operation", "table"})

	dbQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_query_errors_total",
		Help: "Total number of database query errors",
	}, []string{"operation", "table"})
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	db            *gorm.DB
	interval      time.Duration
	stopCh        chan struct{}
	lastWaitCount int64
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(db *gorm.DB, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动指标收集
func (m *MetricsCollector) Start() {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.collect()
			case <-m.stopCh:
				return
			}
		}
	}()
}

// Stop 停止指标收集
func (m *MetricsCollector) Stop() {
	close(m.stopCh)
}

func (m *MetricsCollector) collect() {
	sqlDB, err := m.db.DB()
	if err != nil {
		return
	}
	stats := sqlDB.Stats()
	dbPoolOpenConnections.Set(float64(stats.OpenConnections))
	dbPoolInUseConnections.Set(float64(stats.InUse))
	dbPoolIdleConnections.Set(float64(stats.Idle))

	// 计算增量等待次数
	if stats.WaitCount > m.lastWaitCount {
		dbPoolWaitCount.Add(float64(stats.WaitCount - m.lastWaitCount))
		m.lastWaitCount = stats.WaitCount
	}
}

// RecordQueryDuration 记录查询耗时
func RecordQueryDuration(operation, table string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordQueryError 记录查询错误
func RecordQueryError(operation, table string) {
	dbQueryErrors.WithLabelValues(operation, table).Inc()
}
