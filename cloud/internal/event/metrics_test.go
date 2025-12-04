package event

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProducerMetrics(t *testing.T) {
	metrics := NewProducerMetrics("test")
	assert.NotNil(t, metrics.messagesProduced)
	assert.NotNil(t, metrics.bytesProduced)
	assert.NotNil(t, metrics.produceLatency)
	assert.NotNil(t, metrics.produceErrors)
	assert.NotNil(t, metrics.batchSize)
	assert.NotNil(t, metrics.retryCount)
}

func TestNewProducerMetrics_DefaultNamespace(t *testing.T) {
	metrics := NewProducerMetrics("")
	// Should use "edr" as default namespace
	assert.NotNil(t, metrics)
}

func TestProducerMetrics_Register(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")

	err := metrics.Register(reg)
	require.NoError(t, err)

	// Record something to make metrics appear in Gather
	metrics.RecordMessagesProduced("test-topic", 1, true)

	// Verify metrics are registered
	families, err := reg.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, families)
}

func TestProducerMetrics_RecordMessagesProduced(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordMessagesProduced("test-topic", 10, true)
	metrics.RecordMessagesProduced("test-topic", 5, false)

	// Check counter values
	counter := testutil.ToFloat64(metrics.messagesProduced.WithLabelValues("test-topic", "success"))
	assert.Equal(t, float64(10), counter)

	counter = testutil.ToFloat64(metrics.messagesProduced.WithLabelValues("test-topic", "failure"))
	assert.Equal(t, float64(5), counter)
}

func TestProducerMetrics_RecordBytesProduced(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordBytesProduced("test-topic", 1024)
	metrics.RecordBytesProduced("test-topic", 2048)

	counter := testutil.ToFloat64(metrics.bytesProduced.WithLabelValues("test-topic"))
	assert.Equal(t, float64(3072), counter)
}

func TestProducerMetrics_RecordProduceLatency(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordProduceLatency("test-topic", 0.005)
	metrics.RecordProduceLatency("test-topic", 0.010)

	// For histograms, we check the count via Gather
	families, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, f := range families {
		if f.GetName() == "test_kafka_producer_latency_seconds" {
			for _, m := range f.GetMetric() {
				if m.GetHistogram().GetSampleCount() == 2 {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "should have recorded 2 latency samples")
}

func TestProducerMetrics_RecordProduceError(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordProduceError("test-topic", "timeout")
	metrics.RecordProduceError("test-topic", "timeout")
	metrics.RecordProduceError("test-topic", "network")

	counter := testutil.ToFloat64(metrics.produceErrors.WithLabelValues("test-topic", "timeout"))
	assert.Equal(t, float64(2), counter)

	counter = testutil.ToFloat64(metrics.produceErrors.WithLabelValues("test-topic", "network"))
	assert.Equal(t, float64(1), counter)
}

func TestProducerMetrics_RecordBatchSize(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordBatchSize("test-topic", 50)
	metrics.RecordBatchSize("test-topic", 100)

	// For histograms, we check the count via Gather
	families, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, f := range families {
		if f.GetName() == "test_kafka_producer_batch_size" {
			for _, m := range f.GetMetric() {
				if m.GetHistogram().GetSampleCount() == 2 {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "should have recorded 2 batch size samples")
}

func TestProducerMetrics_RecordRetry(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewProducerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordRetry("test-topic")
	metrics.RecordRetry("test-topic")

	counter := testutil.ToFloat64(metrics.retryCount.WithLabelValues("test-topic"))
	assert.Equal(t, float64(2), counter)
}

func TestNewConsumerMetrics(t *testing.T) {
	metrics := NewConsumerMetrics("test")
	assert.NotNil(t, metrics.messagesConsumed)
	assert.NotNil(t, metrics.bytesConsumed)
	assert.NotNil(t, metrics.consumeLatency)
	assert.NotNil(t, metrics.consumeErrors)
	assert.NotNil(t, metrics.consumerLag)
	assert.NotNil(t, metrics.offsetCommits)
	assert.NotNil(t, metrics.processingTime)
}

func TestConsumerMetrics_Register(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")

	err := metrics.Register(reg)
	require.NoError(t, err)
}

func TestConsumerMetrics_RecordMessagesConsumed(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordMessagesConsumed("test-topic", "test-group", 10, true)
	metrics.RecordMessagesConsumed("test-topic", "test-group", 3, false)

	counter := testutil.ToFloat64(metrics.messagesConsumed.WithLabelValues("test-topic", "test-group", "success"))
	assert.Equal(t, float64(10), counter)

	counter = testutil.ToFloat64(metrics.messagesConsumed.WithLabelValues("test-topic", "test-group", "failure"))
	assert.Equal(t, float64(3), counter)
}

func TestConsumerMetrics_RecordBytesConsumed(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordBytesConsumed("test-topic", "test-group", 1024)

	counter := testutil.ToFloat64(metrics.bytesConsumed.WithLabelValues("test-topic", "test-group"))
	assert.Equal(t, float64(1024), counter)
}

func TestConsumerMetrics_SetConsumerLag(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")
	metrics.MustRegister(reg)

	metrics.SetConsumerLag("test-topic", "test-group", "0", 100)
	metrics.SetConsumerLag("test-topic", "test-group", "1", 50)

	gauge := testutil.ToFloat64(metrics.consumerLag.WithLabelValues("test-topic", "test-group", "0"))
	assert.Equal(t, float64(100), gauge)

	gauge = testutil.ToFloat64(metrics.consumerLag.WithLabelValues("test-topic", "test-group", "1"))
	assert.Equal(t, float64(50), gauge)

	// Update lag
	metrics.SetConsumerLag("test-topic", "test-group", "0", 80)
	gauge = testutil.ToFloat64(metrics.consumerLag.WithLabelValues("test-topic", "test-group", "0"))
	assert.Equal(t, float64(80), gauge)
}

func TestConsumerMetrics_RecordOffsetCommit(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordOffsetCommit("test-topic", "test-group", true)
	metrics.RecordOffsetCommit("test-topic", "test-group", true)
	metrics.RecordOffsetCommit("test-topic", "test-group", false)

	counter := testutil.ToFloat64(metrics.offsetCommits.WithLabelValues("test-topic", "test-group", "success"))
	assert.Equal(t, float64(2), counter)

	counter = testutil.ToFloat64(metrics.offsetCommits.WithLabelValues("test-topic", "test-group", "failure"))
	assert.Equal(t, float64(1), counter)
}

func TestConsumerMetrics_RecordProcessingTime(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewConsumerMetrics("test")
	metrics.MustRegister(reg)

	metrics.RecordProcessingTime("test-topic", "test-group", 0.1)
	metrics.RecordProcessingTime("test-topic", "test-group", 0.2)

	// For histograms, we check the count via Gather
	families, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, f := range families {
		if f.GetName() == "test_kafka_consumer_processing_seconds" {
			for _, m := range f.GetMetric() {
				if m.GetHistogram().GetSampleCount() == 2 {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "should have recorded 2 processing time samples")
}

func TestNewDLQMetrics(t *testing.T) {
	metrics := NewDLQMetrics("test")
	assert.NotNil(t, metrics.messagesRouted)
	assert.NotNil(t, metrics.routeLatency)
	assert.NotNil(t, metrics.routeErrors)
	assert.NotNil(t, metrics.retryAttempts)
}

func TestDLQMetrics_Register(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewDLQMetrics("test")

	err := metrics.Register(reg)
	require.NoError(t, err)
}

func TestDLQMetrics_RecordMessageRouted(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewDLQMetrics("test")
	metrics.Register(reg)

	metrics.RecordMessageRouted("source-topic", "deserialization_error")
	metrics.RecordMessageRouted("source-topic", "processing_error")

	counter := testutil.ToFloat64(metrics.messagesRouted.WithLabelValues("source-topic", "deserialization_error"))
	assert.Equal(t, float64(1), counter)
}

func TestDLQMetrics_RecordRetryAttempt(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewDLQMetrics("test")
	metrics.Register(reg)

	metrics.RecordRetryAttempt("source-topic")
	metrics.RecordRetryAttempt("source-topic")

	counter := testutil.ToFloat64(metrics.retryAttempts.WithLabelValues("source-topic"))
	assert.Equal(t, float64(2), counter)
}

func TestNewHealthMetrics(t *testing.T) {
	metrics := NewHealthMetrics("test")
	assert.NotNil(t, metrics.checkDuration)
	assert.NotNil(t, metrics.checkStatus)
	assert.NotNil(t, metrics.brokersUp)
}

func TestHealthMetrics_Register(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewHealthMetrics("test")

	err := metrics.Register(reg)
	require.NoError(t, err)
}

func TestHealthMetrics_SetCheckStatus(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewHealthMetrics("test")
	metrics.Register(reg)

	metrics.SetCheckStatus("broker", true)
	gauge := testutil.ToFloat64(metrics.checkStatus.WithLabelValues("broker"))
	assert.Equal(t, float64(1), gauge)

	metrics.SetCheckStatus("broker", false)
	gauge = testutil.ToFloat64(metrics.checkStatus.WithLabelValues("broker"))
	assert.Equal(t, float64(0), gauge)
}

func TestHealthMetrics_SetBrokersUp(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewHealthMetrics("test")
	metrics.Register(reg)

	metrics.SetBrokersUp(3)
	gauge := testutil.ToFloat64(metrics.brokersUp.WithLabelValues())
	assert.Equal(t, float64(3), gauge)

	metrics.SetBrokersUp(2)
	gauge = testutil.ToFloat64(metrics.brokersUp.WithLabelValues())
	assert.Equal(t, float64(2), gauge)
}

func TestMetrics_DoubleRegister(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics1 := NewProducerMetrics("test")
	metrics2 := NewProducerMetrics("test")

	err := metrics1.Register(reg)
	require.NoError(t, err)

	// Second registration should fail
	err = metrics2.Register(reg)
	assert.Error(t, err)
}

func TestAllMetrics_Integration(t *testing.T) {
	// Test that all metric types can coexist in the same registry
	reg := prometheus.NewRegistry()

	producerMetrics := NewProducerMetrics("edr")
	consumerMetrics := NewConsumerMetrics("edr")
	dlqMetrics := NewDLQMetrics("edr")
	healthMetrics := NewHealthMetrics("edr")

	err := producerMetrics.Register(reg)
	require.NoError(t, err)

	err = consumerMetrics.Register(reg)
	require.NoError(t, err)

	err = dlqMetrics.Register(reg)
	require.NoError(t, err)

	err = healthMetrics.Register(reg)
	require.NoError(t, err)

	// Record some metrics
	producerMetrics.RecordMessagesProduced("edr.events.raw", 100, true)
	consumerMetrics.RecordMessagesConsumed("edr.events.raw", "event-processor", 100, true)
	dlqMetrics.RecordMessageRouted("edr.events.raw", "processing_error")
	healthMetrics.SetCheckStatus("broker", true)

	// Gather all metrics
	families, err := reg.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, families)
}
