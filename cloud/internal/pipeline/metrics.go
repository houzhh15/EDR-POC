// Package pipeline provides Prometheus metrics for the event processing pipeline.
package pipeline

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PipelineMetrics 管线指标
type PipelineMetrics struct {
	eventsConsumed     *prometheus.CounterVec
	eventsProcessed    *prometheus.CounterVec
	eventsEnriched     *prometheus.CounterVec
	eventsWritten      *prometheus.CounterVec
	processingDuration *prometheus.HistogramVec
	batchSize          *prometheus.HistogramVec
	errorsTotal        *prometheus.CounterVec
	dlqMessages        *prometheus.CounterVec
	consumerLag        *prometheus.GaugeVec
	enricherLatency    *prometheus.HistogramVec
	writerLatency      *prometheus.HistogramVec
	bufferSize         *prometheus.GaugeVec
}

// NewPipelineMetrics 创建新的管线指标
func NewPipelineMetrics(namespace string) *PipelineMetrics {
	if namespace == "" {
		namespace = "edr"
	}

	return &PipelineMetrics{
		eventsConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "events_consumed_total",
				Help:      "Total events consumed from Kafka",
			},
			[]string{"topic"},
		),
		eventsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "events_processed_total",
				Help:      "Total events processed",
			},
			[]string{"event_type", "status"},
		),
		eventsEnriched: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "events_enriched_total",
				Help:      "Total events enriched",
			},
			[]string{"enricher", "status"},
		),
		eventsWritten: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "events_written_total",
				Help:      "Total events written to outputs",
			},
			[]string{"output", "status"},
		),
		processingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "processing_duration_seconds",
				Help:      "Processing duration by stage",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"stage"},
		),
		batchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "batch_size",
				Help:      "Batch size histogram",
				Buckets:   []float64{1, 10, 50, 100, 250, 500, 1000, 2000},
			},
			[]string{"output"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "errors_total",
				Help:      "Total errors by stage and type",
			},
			[]string{"stage", "error_type"},
		),
		dlqMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "dlq_messages_total",
				Help:      "Total messages sent to DLQ",
			},
			[]string{"reason"},
		),
		consumerLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "consumer_lag",
				Help:      "Consumer lag by partition",
			},
			[]string{"topic", "partition"},
		),
		enricherLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "enricher_latency_seconds",
				Help:      "Enricher latency in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .05, .1},
			},
			[]string{"enricher"},
		),
		writerLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "writer_latency_seconds",
				Help:      "Writer latency in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2},
			},
			[]string{"writer"},
		),
		bufferSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "pipeline",
				Name:      "buffer_size",
				Help:      "Current buffer size",
			},
			[]string{"buffer"},
		),
	}
}

// Register 注册所有指标
func (m *PipelineMetrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.eventsConsumed,
		m.eventsProcessed,
		m.eventsEnriched,
		m.eventsWritten,
		m.processingDuration,
		m.batchSize,
		m.errorsTotal,
		m.dlqMessages,
		m.consumerLag,
		m.enricherLatency,
		m.writerLatency,
		m.bufferSize,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// MustRegister 注册所有指标（失败时 panic）
func (m *PipelineMetrics) MustRegister(reg prometheus.Registerer) {
	reg.MustRegister(
		m.eventsConsumed,
		m.eventsProcessed,
		m.eventsEnriched,
		m.eventsWritten,
		m.processingDuration,
		m.batchSize,
		m.errorsTotal,
		m.dlqMessages,
		m.consumerLag,
		m.enricherLatency,
		m.writerLatency,
		m.bufferSize,
	)
}

// RecordEventConsumed 记录消费事件
func (m *PipelineMetrics) RecordEventConsumed(topic string, count int) {
	m.eventsConsumed.WithLabelValues(topic).Add(float64(count))
}

// RecordEventProcessed 记录处理事件
func (m *PipelineMetrics) RecordEventProcessed(eventType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.eventsProcessed.WithLabelValues(eventType, status).Inc()
}

// RecordEventEnriched 记录丰富化事件
func (m *PipelineMetrics) RecordEventEnriched(enricher string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.eventsEnriched.WithLabelValues(enricher, status).Inc()
}

// RecordEventWritten 记录写入事件
func (m *PipelineMetrics) RecordEventWritten(output string, count int, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.eventsWritten.WithLabelValues(output, status).Add(float64(count))
}

// RecordProcessingDuration 记录处理耗时
func (m *PipelineMetrics) RecordProcessingDuration(stage string, seconds float64) {
	m.processingDuration.WithLabelValues(stage).Observe(seconds)
}

// RecordBatchSize 记录批量大小
func (m *PipelineMetrics) RecordBatchSize(output string, size int) {
	m.batchSize.WithLabelValues(output).Observe(float64(size))
}

// RecordError 记录错误
func (m *PipelineMetrics) RecordError(stage, errorType string) {
	m.errorsTotal.WithLabelValues(stage, errorType).Inc()
}

// RecordDLQMessage 记录 DLQ 消息
func (m *PipelineMetrics) RecordDLQMessage(reason string) {
	m.dlqMessages.WithLabelValues(reason).Inc()
}

// SetConsumerLag 设置消费者延迟
func (m *PipelineMetrics) SetConsumerLag(topic string, partition int, lag float64) {
	m.consumerLag.WithLabelValues(topic, string(rune('0'+partition))).Set(lag)
}

// RecordEnricherLatency 记录丰富化器延迟
func (m *PipelineMetrics) RecordEnricherLatency(enricher string, seconds float64) {
	m.enricherLatency.WithLabelValues(enricher).Observe(seconds)
}

// RecordWriterLatency 记录写入器延迟
func (m *PipelineMetrics) RecordWriterLatency(writer string, seconds float64) {
	m.writerLatency.WithLabelValues(writer).Observe(seconds)
}

// SetBufferSize 设置缓冲区大小
func (m *PipelineMetrics) SetBufferSize(buffer string, size int) {
	m.bufferSize.WithLabelValues(buffer).Set(float64(size))
}
