// Package event provides Prometheus metrics for Kafka components.
package event

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ProducerMetrics contains Prometheus metrics for Kafka producer.
type ProducerMetrics struct {
	messagesProduced *prometheus.CounterVec
	bytesProduced    *prometheus.CounterVec
	produceLatency   *prometheus.HistogramVec
	produceErrors    *prometheus.CounterVec
	batchSize        *prometheus.HistogramVec
	retryCount       *prometheus.CounterVec
}

// NewProducerMetrics creates a new ProducerMetrics instance.
func NewProducerMetrics(namespace string) *ProducerMetrics {
	if namespace == "" {
		namespace = "edr"
	}

	return &ProducerMetrics{
		messagesProduced: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "messages_total",
				Help:      "Total number of messages produced",
			},
			[]string{"topic", "status"},
		),
		bytesProduced: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "bytes_total",
				Help:      "Total bytes produced",
			},
			[]string{"topic"},
		),
		produceLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "latency_seconds",
				Help:      "Produce latency in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"topic"},
		),
		produceErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "errors_total",
				Help:      "Total produce errors",
			},
			[]string{"topic", "error_type"},
		),
		batchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "batch_size",
				Help:      "Batch size histogram",
				Buckets:   []float64{1, 10, 50, 100, 500, 1000},
			},
			[]string{"topic"},
		),
		retryCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_producer",
				Name:      "retries_total",
				Help:      "Total retry attempts",
			},
			[]string{"topic"},
		),
	}
}

// Register registers all metrics with the given registerer.
func (m *ProducerMetrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.messagesProduced,
		m.bytesProduced,
		m.produceLatency,
		m.produceErrors,
		m.batchSize,
		m.retryCount,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// MustRegister registers all metrics and panics on error.
func (m *ProducerMetrics) MustRegister(reg prometheus.Registerer) {
	reg.MustRegister(
		m.messagesProduced,
		m.bytesProduced,
		m.produceLatency,
		m.produceErrors,
		m.batchSize,
		m.retryCount,
	)
}

// RecordMessagesProduced records successful or failed message production.
func (m *ProducerMetrics) RecordMessagesProduced(topic string, count int, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.messagesProduced.WithLabelValues(topic, status).Add(float64(count))
}

// RecordBytesProduced records bytes produced to a topic.
func (m *ProducerMetrics) RecordBytesProduced(topic string, bytes int) {
	m.bytesProduced.WithLabelValues(topic).Add(float64(bytes))
}

// RecordProduceLatency records produce latency in seconds.
func (m *ProducerMetrics) RecordProduceLatency(topic string, seconds float64) {
	m.produceLatency.WithLabelValues(topic).Observe(seconds)
}

// RecordProduceError records a produce error.
func (m *ProducerMetrics) RecordProduceError(topic, errorType string) {
	m.produceErrors.WithLabelValues(topic, errorType).Inc()
}

// RecordBatchSize records the batch size.
func (m *ProducerMetrics) RecordBatchSize(topic string, size int) {
	m.batchSize.WithLabelValues(topic).Observe(float64(size))
}

// RecordRetry records a retry attempt.
func (m *ProducerMetrics) RecordRetry(topic string) {
	m.retryCount.WithLabelValues(topic).Inc()
}

// ConsumerMetrics contains Prometheus metrics for Kafka consumer.
type ConsumerMetrics struct {
	messagesConsumed *prometheus.CounterVec
	bytesConsumed    *prometheus.CounterVec
	consumeLatency   *prometheus.HistogramVec
	consumeErrors    *prometheus.CounterVec
	consumerLag      *prometheus.GaugeVec
	offsetCommits    *prometheus.CounterVec
	processingTime   *prometheus.HistogramVec
}

// NewConsumerMetrics creates a new ConsumerMetrics instance.
func NewConsumerMetrics(namespace string) *ConsumerMetrics {
	if namespace == "" {
		namespace = "edr"
	}

	return &ConsumerMetrics{
		messagesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "messages_total",
				Help:      "Total messages consumed",
			},
			[]string{"topic", "group", "status"},
		),
		bytesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "bytes_total",
				Help:      "Total bytes consumed",
			},
			[]string{"topic", "group"},
		),
		consumeLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "latency_seconds",
				Help:      "Message consume latency (time from message timestamp to processing)",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"topic", "group"},
		),
		consumeErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "errors_total",
				Help:      "Total consume errors",
			},
			[]string{"topic", "group", "error_type"},
		),
		consumerLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "lag",
				Help:      "Consumer group lag",
			},
			[]string{"topic", "group", "partition"},
		),
		offsetCommits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "commits_total",
				Help:      "Total offset commits",
			},
			[]string{"topic", "group", "status"},
		),
		processingTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_consumer",
				Name:      "processing_seconds",
				Help:      "Message processing time in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"topic", "group"},
		),
	}
}

// Register registers all metrics with the given registerer.
func (m *ConsumerMetrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.messagesConsumed,
		m.bytesConsumed,
		m.consumeLatency,
		m.consumeErrors,
		m.consumerLag,
		m.offsetCommits,
		m.processingTime,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// MustRegister registers all metrics and panics on error.
func (m *ConsumerMetrics) MustRegister(reg prometheus.Registerer) {
	reg.MustRegister(
		m.messagesConsumed,
		m.bytesConsumed,
		m.consumeLatency,
		m.consumeErrors,
		m.consumerLag,
		m.offsetCommits,
		m.processingTime,
	)
}

// RecordMessagesConsumed records successful or failed message consumption.
func (m *ConsumerMetrics) RecordMessagesConsumed(topic, group string, count int, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.messagesConsumed.WithLabelValues(topic, group, status).Add(float64(count))
}

// RecordBytesConsumed records bytes consumed from a topic.
func (m *ConsumerMetrics) RecordBytesConsumed(topic, group string, bytes int) {
	m.bytesConsumed.WithLabelValues(topic, group).Add(float64(bytes))
}

// RecordConsumeLatency records consume latency in seconds.
func (m *ConsumerMetrics) RecordConsumeLatency(topic, group string, seconds float64) {
	m.consumeLatency.WithLabelValues(topic, group).Observe(seconds)
}

// RecordConsumeError records a consume error.
func (m *ConsumerMetrics) RecordConsumeError(topic, group, errorType string) {
	m.consumeErrors.WithLabelValues(topic, group, errorType).Inc()
}

// SetConsumerLag sets the consumer lag for a partition.
func (m *ConsumerMetrics) SetConsumerLag(topic, group, partition string, lag float64) {
	m.consumerLag.WithLabelValues(topic, group, partition).Set(lag)
}

// RecordOffsetCommit records an offset commit (success or failure).
func (m *ConsumerMetrics) RecordOffsetCommit(topic, group string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.offsetCommits.WithLabelValues(topic, group, status).Inc()
}

// RecordProcessingTime records message processing time in seconds.
func (m *ConsumerMetrics) RecordProcessingTime(topic, group string, seconds float64) {
	m.processingTime.WithLabelValues(topic, group).Observe(seconds)
}

// DLQMetrics contains Prometheus metrics for Dead Letter Queue.
type DLQMetrics struct {
	messagesRouted *prometheus.CounterVec
	routeLatency   *prometheus.HistogramVec
	routeErrors    *prometheus.CounterVec
	retryAttempts  *prometheus.CounterVec
}

// NewDLQMetrics creates a new DLQMetrics instance.
func NewDLQMetrics(namespace string) *DLQMetrics {
	if namespace == "" {
		namespace = "edr"
	}

	return &DLQMetrics{
		messagesRouted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_dlq",
				Name:      "messages_total",
				Help:      "Total messages routed to DLQ",
			},
			[]string{"original_topic", "reason"},
		),
		routeLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_dlq",
				Name:      "route_latency_seconds",
				Help:      "DLQ routing latency in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"original_topic"},
		),
		routeErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_dlq",
				Name:      "route_errors_total",
				Help:      "Total DLQ routing errors",
			},
			[]string{"original_topic", "error_type"},
		),
		retryAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka_dlq",
				Name:      "retry_attempts_total",
				Help:      "Total retry attempts before DLQ",
			},
			[]string{"original_topic"},
		),
	}
}

// Register registers all metrics with the given registerer.
func (m *DLQMetrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.messagesRouted,
		m.routeLatency,
		m.routeErrors,
		m.retryAttempts,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RecordMessageRouted records a message routed to DLQ.
func (m *DLQMetrics) RecordMessageRouted(originalTopic, reason string) {
	m.messagesRouted.WithLabelValues(originalTopic, reason).Inc()
}

// RecordRouteLatency records DLQ routing latency.
func (m *DLQMetrics) RecordRouteLatency(originalTopic string, seconds float64) {
	m.routeLatency.WithLabelValues(originalTopic).Observe(seconds)
}

// RecordRouteError records a DLQ routing error.
func (m *DLQMetrics) RecordRouteError(originalTopic, errorType string) {
	m.routeErrors.WithLabelValues(originalTopic, errorType).Inc()
}

// RecordRetryAttempt records a retry attempt.
func (m *DLQMetrics) RecordRetryAttempt(originalTopic string) {
	m.retryAttempts.WithLabelValues(originalTopic).Inc()
}

// HealthMetrics contains Prometheus metrics for health checks.
type HealthMetrics struct {
	checkDuration *prometheus.HistogramVec
	checkStatus   *prometheus.GaugeVec
	brokersUp     *prometheus.GaugeVec
}

// NewHealthMetrics creates a new HealthMetrics instance.
func NewHealthMetrics(namespace string) *HealthMetrics {
	if namespace == "" {
		namespace = "edr"
	}

	return &HealthMetrics{
		checkDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "kafka_health",
				Name:      "check_duration_seconds",
				Help:      "Health check duration in seconds",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"check_type"},
		),
		checkStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "kafka_health",
				Name:      "status",
				Help:      "Health check status (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),
		brokersUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "kafka_health",
				Name:      "brokers_up",
				Help:      "Number of healthy brokers",
			},
			[]string{},
		),
	}
}

// Register registers all metrics with the given registerer.
func (m *HealthMetrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.checkDuration,
		m.checkStatus,
		m.brokersUp,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RecordCheckDuration records health check duration.
func (m *HealthMetrics) RecordCheckDuration(checkType string, seconds float64) {
	m.checkDuration.WithLabelValues(checkType).Observe(seconds)
}

// SetCheckStatus sets the health check status.
func (m *HealthMetrics) SetCheckStatus(component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.checkStatus.WithLabelValues(component).Set(value)
}

// SetBrokersUp sets the number of healthy brokers.
func (m *HealthMetrics) SetBrokersUp(count int) {
	m.brokersUp.WithLabelValues().Set(float64(count))
}
