// Package event provides Kafka health checking functionality.
package event

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// HealthChecker checks Kafka broker and topic health.
type HealthChecker struct {
	brokers []string
	timeout time.Duration
	logger  *zap.Logger
	metrics *HealthMetrics
}

// HealthCheckerConfig configuration for HealthChecker.
type HealthCheckerConfig struct {
	Brokers       []string      `yaml:"brokers"`
	Timeout       time.Duration `yaml:"timeout"`
	CheckInterval time.Duration `yaml:"check_interval"`
}

// DefaultHealthCheckerConfig returns default health checker configuration.
func DefaultHealthCheckerConfig() *HealthCheckerConfig {
	return &HealthCheckerConfig{
		Brokers:       []string{"localhost:19092"},
		Timeout:       5 * time.Second,
		CheckInterval: 30 * time.Second,
	}
}

// HealthStatus represents the overall health status.
type HealthStatus struct {
	Healthy   bool           `json:"healthy"`
	Brokers   []BrokerStatus `json:"brokers"`
	Topics    []TopicStatus  `json:"topics,omitempty"`
	CheckedAt time.Time      `json:"checked_at"`
	Error     string         `json:"error,omitempty"`
	Duration  string         `json:"duration"`
}

// BrokerStatus represents the health status of a single broker.
type BrokerStatus struct {
	Address string `json:"address"`
	Healthy bool   `json:"healthy"`
	Latency string `json:"latency"`
	Error   string `json:"error,omitempty"`
}

// TopicStatus represents the health status of a topic.
type TopicStatus struct {
	Name       string `json:"name"`
	Partitions int    `json:"partitions"`
	Healthy    bool   `json:"healthy"`
	Error      string `json:"error,omitempty"`
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(brokers []string, timeout time.Duration, logger *zap.Logger) *HealthChecker {
	if len(brokers) == 0 {
		brokers = []string{"localhost:19092"}
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HealthChecker{
		brokers: brokers,
		timeout: timeout,
		logger:  logger,
	}
}

// SetMetrics sets the health metrics collector.
func (h *HealthChecker) SetMetrics(metrics *HealthMetrics) {
	h.metrics = metrics
}

// Check performs a comprehensive health check.
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	start := time.Now()

	status := &HealthStatus{
		Healthy:   true,
		CheckedAt: start,
	}

	// Check brokers
	brokerStatuses := h.CheckBrokers(ctx)
	status.Brokers = brokerStatuses

	// Count healthy brokers
	healthyCount := 0
	for _, b := range brokerStatuses {
		if b.Healthy {
			healthyCount++
		}
	}

	// Consider unhealthy if no brokers are available
	if healthyCount == 0 {
		status.Healthy = false
		status.Error = "no healthy brokers available"
	}

	status.Duration = time.Since(start).String()

	if h.metrics != nil {
		h.metrics.RecordCheckDuration("full", time.Since(start).Seconds())
		h.metrics.SetCheckStatus("kafka", status.Healthy)
		h.metrics.SetBrokersUp(healthyCount)
	}

	return status
}

// CheckBrokers checks the connectivity of all brokers.
func (h *HealthChecker) CheckBrokers(ctx context.Context) []BrokerStatus {
	var wg sync.WaitGroup
	results := make([]BrokerStatus, len(h.brokers))

	for i, broker := range h.brokers {
		wg.Add(1)
		go func(idx int, addr string) {
			defer wg.Done()
			results[idx] = h.checkBroker(ctx, addr)
		}(i, broker)
	}

	wg.Wait()
	return results
}

// checkBroker checks a single broker's health.
func (h *HealthChecker) checkBroker(ctx context.Context, addr string) BrokerStatus {
	status := BrokerStatus{
		Address: addr,
		Healthy: false,
	}

	start := time.Now()

	// Create a context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Try to connect
	dialer := &kafka.Dialer{
		Timeout:   h.timeout,
		DualStack: true,
	}

	conn, err := dialer.DialContext(checkCtx, "tcp", addr)
	if err != nil {
		status.Error = fmt.Sprintf("connection failed: %v", err)
		status.Latency = "N/A"
		h.logger.Debug("broker health check failed",
			zap.String("broker", addr),
			zap.Error(err),
		)
		return status
	}
	defer conn.Close()

	// Try to get broker info
	_, err = conn.Brokers()
	if err != nil {
		status.Error = fmt.Sprintf("failed to get broker info: %v", err)
		status.Latency = time.Since(start).String()
		return status
	}

	status.Healthy = true
	status.Latency = time.Since(start).String()

	h.logger.Debug("broker health check passed",
		zap.String("broker", addr),
		zap.Duration("latency", time.Since(start)),
	)

	return status
}

// CheckTopics checks the availability of specified topics.
func (h *HealthChecker) CheckTopics(ctx context.Context, topics []string) []TopicStatus {
	results := make([]TopicStatus, 0, len(topics))

	// Get a connection to check topics
	var conn *kafka.Conn
	var connErr error

	for _, broker := range h.brokers {
		checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
		dialer := &kafka.Dialer{
			Timeout:   h.timeout,
			DualStack: true,
		}
		conn, connErr = dialer.DialContext(checkCtx, "tcp", broker)
		cancel()
		if connErr == nil {
			break
		}
	}

	if connErr != nil {
		// Return all topics as unhealthy
		for _, topic := range topics {
			results = append(results, TopicStatus{
				Name:    topic,
				Healthy: false,
				Error:   fmt.Sprintf("no broker connection: %v", connErr),
			})
		}
		return results
	}
	defer conn.Close()

	// Get controller for topic operations
	controller, err := conn.Controller()
	if err != nil {
		for _, topic := range topics {
			results = append(results, TopicStatus{
				Name:    topic,
				Healthy: false,
				Error:   fmt.Sprintf("failed to get controller: %v", err),
			})
		}
		return results
	}

	// Connect to controller
	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	dialer := &kafka.Dialer{
		Timeout:   h.timeout,
		DualStack: true,
	}

	controllerCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	controllerConn, err := dialer.DialContext(controllerCtx, "tcp", controllerAddr)
	if err != nil {
		for _, topic := range topics {
			results = append(results, TopicStatus{
				Name:    topic,
				Healthy: false,
				Error:   fmt.Sprintf("failed to connect to controller: %v", err),
			})
		}
		return results
	}
	defer controllerConn.Close()

	// Check each topic
	for _, topic := range topics {
		status := h.checkTopic(ctx, controllerConn, topic)
		results = append(results, status)
	}

	return results
}

// checkTopic checks a single topic's health.
func (h *HealthChecker) checkTopic(ctx context.Context, conn *kafka.Conn, topic string) TopicStatus {
	status := TopicStatus{
		Name:    topic,
		Healthy: false,
	}

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		status.Error = fmt.Sprintf("failed to read partitions: %v", err)
		h.logger.Debug("topic health check failed",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return status
	}

	if len(partitions) == 0 {
		status.Error = "no partitions found"
		return status
	}

	status.Healthy = true
	status.Partitions = len(partitions)

	h.logger.Debug("topic health check passed",
		zap.String("topic", topic),
		zap.Int("partitions", len(partitions)),
	)

	return status
}

// CheckWithTopics performs a health check including topic validation.
func (h *HealthChecker) CheckWithTopics(ctx context.Context, topics []string) *HealthStatus {
	status := h.Check(ctx)

	if len(topics) > 0 && status.Healthy {
		topicStatuses := h.CheckTopics(ctx, topics)
		status.Topics = topicStatuses

		// Check if any required topic is unhealthy
		for _, ts := range topicStatuses {
			if !ts.Healthy {
				status.Healthy = false
				status.Error = fmt.Sprintf("topic %s is unhealthy: %s", ts.Name, ts.Error)
				break
			}
		}
	}

	return status
}

// Ping performs a simple connectivity check.
func (h *HealthChecker) Ping(ctx context.Context) error {
	for _, broker := range h.brokers {
		checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
		dialer := &kafka.Dialer{
			Timeout:   h.timeout,
			DualStack: true,
		}
		conn, err := dialer.DialContext(checkCtx, "tcp", broker)
		cancel()
		if err == nil {
			conn.Close()
			return nil
		}
	}
	return fmt.Errorf("failed to connect to any broker")
}

// IsHealthy returns true if Kafka is healthy.
func (h *HealthChecker) IsHealthy(ctx context.Context) bool {
	return h.Check(ctx).Healthy
}
