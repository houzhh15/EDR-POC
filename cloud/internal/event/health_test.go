package event

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestDefaultHealthCheckerConfig(t *testing.T) {
	cfg := DefaultHealthCheckerConfig()

	assert.Equal(t, []string{"localhost:19092"}, cfg.Brokers)
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, 30*time.Second, cfg.CheckInterval)
}

func TestNewHealthChecker(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
		timeout time.Duration
	}{
		{
			name:    "valid brokers and timeout",
			brokers: []string{"localhost:9092"},
			timeout: 5 * time.Second,
		},
		{
			name:    "nil brokers uses defaults",
			brokers: nil,
			timeout: 5 * time.Second,
		},
		{
			name:    "empty brokers uses defaults",
			brokers: []string{},
			timeout: 5 * time.Second,
		},
		{
			name:    "zero timeout uses default",
			brokers: []string{"localhost:9092"},
			timeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hc := NewHealthChecker(tt.brokers, tt.timeout, nil)
			assert.NotNil(t, hc)
		})
	}
}

func TestHealthStatus_Fields(t *testing.T) {
	status := &HealthStatus{
		Healthy:   true,
		CheckedAt: time.Now(),
		Duration:  "100ms",
		Brokers: []BrokerStatus{
			{Address: "localhost:9092", Healthy: true, Latency: "50ms"},
		},
		Topics: []TopicStatus{
			{Name: "test-topic", Healthy: true, Partitions: 3},
		},
	}

	assert.True(t, status.Healthy)
	assert.Len(t, status.Brokers, 1)
	assert.Len(t, status.Topics, 1)
	assert.NotEmpty(t, status.Duration)
}

func TestBrokerStatus(t *testing.T) {
	bs := BrokerStatus{
		Address: "localhost:9092",
		Healthy: true,
		Latency: "50ms",
		Error:   "",
	}

	assert.Equal(t, "localhost:9092", bs.Address)
	assert.True(t, bs.Healthy)
	assert.Equal(t, "50ms", bs.Latency)
	assert.Empty(t, bs.Error)
}

func TestTopicStatus(t *testing.T) {
	ts := TopicStatus{
		Name:       "test-topic",
		Healthy:    true,
		Partitions: 3,
		Error:      "",
	}

	assert.Equal(t, "test-topic", ts.Name)
	assert.True(t, ts.Healthy)
	assert.Equal(t, 3, ts.Partitions)
	assert.Empty(t, ts.Error)
}

func TestHealthChecker_CheckBrokers_InvalidBroker(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Use an invalid address that will fail fast
	brokers := []string{"invalid-host-that-does-not-exist:9092"}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	brokerStatuses := hc.CheckBrokers(ctx)

	assert.Len(t, brokerStatuses, 1)
	assert.Equal(t, "invalid-host-that-does-not-exist:9092", brokerStatuses[0].Address)
	assert.False(t, brokerStatuses[0].Healthy)
	assert.NotEmpty(t, brokerStatuses[0].Error)
}

func TestHealthChecker_Check_NoHealthyBrokers(t *testing.T) {
	logger := zaptest.NewLogger(t)

	brokers := []string{"invalid-broker:9092"}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	status := hc.Check(ctx)

	assert.NotNil(t, status)
	assert.False(t, status.Healthy)
	assert.NotEmpty(t, status.Error)
}

func TestHealthChecker_Ping_NoBrokers(t *testing.T) {
	logger := zaptest.NewLogger(t)

	brokers := []string{"invalid-broker:9092"}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	err := hc.Ping(ctx)

	assert.Error(t, err)
}

func TestHealthChecker_IsHealthy_False(t *testing.T) {
	logger := zaptest.NewLogger(t)

	brokers := []string{"invalid-broker:9092"}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	healthy := hc.IsHealthy(ctx)

	assert.False(t, healthy)
}

func TestHealthChecker_SetMetrics(t *testing.T) {
	hc := NewHealthChecker([]string{"localhost:9092"}, 5*time.Second, nil)

	metrics := NewHealthMetrics("edr_test")
	hc.SetMetrics(metrics)

	// Metrics should be set without panic
}

// Integration tests - only run when Kafka is available
func TestHealthChecker_Check_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	status := hc.Check(ctx)

	assert.NotNil(t, status)
	assert.NotEmpty(t, status.CheckedAt)

	// Log detailed status for debugging
	t.Logf("Health status: healthy=%v, duration=%s", status.Healthy, status.Duration)
	for _, bs := range status.Brokers {
		t.Logf("  Broker %s: healthy=%v, latency=%s, error=%s", bs.Address, bs.Healthy, bs.Latency, bs.Error)
	}
}

func TestHealthChecker_CheckBrokers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	brokerStatuses := hc.CheckBrokers(ctx)

	require.Len(t, brokerStatuses, 1)
	assert.Equal(t, "localhost:19092", brokerStatuses[0].Address)
	assert.True(t, brokerStatuses[0].Healthy)
	assert.Empty(t, brokerStatuses[0].Error)
	assert.NotEmpty(t, brokerStatuses[0].Latency)
}

func TestHealthChecker_CheckTopics_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	topicStatuses := hc.CheckTopics(ctx, []string{"edr.events.raw"})

	require.Len(t, topicStatuses, 1)
	assert.Equal(t, "edr.events.raw", topicStatuses[0].Name)
	// Topic may or may not exist depending on test environment
	t.Logf("Topic edr.events.raw: healthy=%v, partitions=%d, error=%s",
		topicStatuses[0].Healthy, topicStatuses[0].Partitions, topicStatuses[0].Error)
}

func TestHealthChecker_CheckWithTopics_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	status := hc.CheckWithTopics(ctx, []string{"edr.events.raw"})

	assert.NotNil(t, status)
	t.Logf("Health status with topics: healthy=%v", status.Healthy)
}

func TestHealthChecker_Ping_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	err = hc.Ping(ctx)
	assert.NoError(t, err)
}

func TestHealthChecker_IsHealthy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if Kafka/Redpanda is available
	conn, err := net.DialTimeout("tcp", "localhost:19092", time.Second)
	if err != nil {
		t.Skip("Kafka not available at localhost:19092")
	}
	conn.Close()

	logger := zaptest.NewLogger(t)
	brokers := []string{"localhost:19092"}
	hc := NewHealthChecker(brokers, 5*time.Second, logger)

	ctx := context.Background()
	healthy := hc.IsHealthy(ctx)
	assert.True(t, healthy)
}

func TestHealthStatus_WithErrors(t *testing.T) {
	status := &HealthStatus{
		Healthy:   false,
		CheckedAt: time.Now(),
		Duration:  "200ms",
		Error:     "partial failure",
		Brokers: []BrokerStatus{
			{Address: "localhost:9092", Healthy: true, Latency: "50ms"},
			{Address: "localhost:9093", Healthy: false, Latency: "N/A", Error: "connection refused"},
			{Address: "localhost:9094", Healthy: false, Latency: "N/A", Error: "timeout"},
		},
		Topics: []TopicStatus{
			{Name: "topic1", Healthy: true, Partitions: 3},
			{Name: "topic2", Healthy: true, Partitions: 1},
			{Name: "topic3", Healthy: false, Error: "not found"},
		},
	}

	assert.False(t, status.Healthy)
	assert.Equal(t, "partial failure", status.Error)
	assert.Len(t, status.Brokers, 3)
	assert.Len(t, status.Topics, 3)

	// Count healthy
	healthyBrokers := 0
	for _, b := range status.Brokers {
		if b.Healthy {
			healthyBrokers++
		}
	}
	assert.Equal(t, 1, healthyBrokers)

	healthyTopics := 0
	for _, ts := range status.Topics {
		if ts.Healthy {
			healthyTopics++
		}
	}
	assert.Equal(t, 2, healthyTopics)
}

func TestHealthChecker_MultipleBrokers(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Multiple invalid brokers
	brokers := []string{
		"invalid-broker1:9092",
		"invalid-broker2:9092",
		"invalid-broker3:9092",
	}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	brokerStatuses := hc.CheckBrokers(ctx)

	assert.Len(t, brokerStatuses, 3)
	for _, bs := range brokerStatuses {
		assert.False(t, bs.Healthy)
		assert.NotEmpty(t, bs.Error)
	}
}

// Test concurrent broker checking
func TestHealthChecker_ConcurrentBrokerChecks(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Multiple brokers - should be checked concurrently
	brokers := []string{
		"invalid-broker1:9092",
		"invalid-broker2:9092",
		"invalid-broker3:9092",
		"invalid-broker4:9092",
		"invalid-broker5:9092",
	}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()

	start := time.Now()
	brokerStatuses := hc.CheckBrokers(ctx)
	elapsed := time.Since(start)

	assert.Len(t, brokerStatuses, 5)

	// If concurrent, should complete in roughly 1 timeout period
	// If sequential, would take 5x the timeout
	// Allow some margin for overhead
	assert.Less(t, elapsed, 500*time.Millisecond,
		"Broker checks should be concurrent, not sequential")
}

func TestHealthChecker_CheckTopics_NoBrokerConnection(t *testing.T) {
	logger := zaptest.NewLogger(t)

	brokers := []string{"invalid-broker:9092"}
	hc := NewHealthChecker(brokers, 100*time.Millisecond, logger)

	ctx := context.Background()
	topicStatuses := hc.CheckTopics(ctx, []string{"topic1", "topic2"})

	assert.Len(t, topicStatuses, 2)
	for _, ts := range topicStatuses {
		assert.False(t, ts.Healthy)
		assert.Contains(t, ts.Error, "no broker connection")
	}
}
