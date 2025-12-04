// Package event provides Kafka messaging components for EDR cloud services.
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

// TopicManager manages Kafka topics creation and configuration.
type TopicManager struct {
	brokers    []string
	config     *TopicManagerConfig
	logger     *zap.Logger
	mu         sync.Mutex
	controller string // cached controller address
}

// TopicManagerConfig configuration for TopicManager.
type TopicManagerConfig struct {
	Brokers      []string                   `yaml:"brokers"`
	TopicConfigs map[string]TopicDefinition `yaml:"topics"`
	AutoCreate   bool                       `yaml:"auto_create"`
	DialTimeout  time.Duration              `yaml:"dial_timeout"`
}

// TopicDefinition defines a single topic's configuration.
type TopicDefinition struct {
	Partitions        int    `yaml:"partitions"`
	ReplicationFactor int    `yaml:"replication_factor"`
	RetentionMs       int64  `yaml:"retention_ms"`
	CleanupPolicy     string `yaml:"cleanup_policy"` // delete, compact
}

// DefaultTopicConfigs returns default topic configurations for EDR.
func DefaultTopicConfigs() map[string]TopicDefinition {
	return map[string]TopicDefinition{
		"edr.events.raw": {
			Partitions:        12,
			ReplicationFactor: 1,                       // dev environment single replica
			RetentionMs:       7 * 24 * 60 * 60 * 1000, // 7 days
			CleanupPolicy:     "delete",
		},
		"edr.events.normalized": {
			Partitions:        12,
			ReplicationFactor: 1,
			RetentionMs:       7 * 24 * 60 * 60 * 1000,
			CleanupPolicy:     "delete",
		},
		"edr.alerts": {
			Partitions:        6,
			ReplicationFactor: 1,
			RetentionMs:       30 * 24 * 60 * 60 * 1000, // 30 days
			CleanupPolicy:     "delete",
		},
		"edr.commands": {
			Partitions:        6,
			ReplicationFactor: 1,
			RetentionMs:       24 * 60 * 60 * 1000, // 1 day
			CleanupPolicy:     "delete",
		},
		"edr.dlq": {
			Partitions:        3,
			ReplicationFactor: 1,
			RetentionMs:       30 * 24 * 60 * 60 * 1000,
			CleanupPolicy:     "delete",
		},
	}
}

// DefaultTopicManagerConfig returns default TopicManager configuration.
func DefaultTopicManagerConfig() *TopicManagerConfig {
	return &TopicManagerConfig{
		Brokers:      []string{"localhost:19092"},
		TopicConfigs: DefaultTopicConfigs(),
		AutoCreate:   true,
		DialTimeout:  10 * time.Second,
	}
}

// NewTopicManager creates a new TopicManager.
func NewTopicManager(cfg *TopicManagerConfig, logger *zap.Logger) (*TopicManager, error) {
	if cfg == nil {
		cfg = DefaultTopicManagerConfig()
	}
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 10 * time.Second
	}

	return &TopicManager{
		brokers: cfg.Brokers,
		config:  cfg,
		logger:  logger,
	}, nil
}

// getController gets the controller broker address.
func (m *TopicManager) getController(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.controller != "" {
		return m.controller, nil
	}

	// Try each broker to find the controller
	for _, broker := range m.brokers {
		conn, err := m.dialBroker(ctx, broker)
		if err != nil {
			m.logger.Debug("failed to connect to broker", zap.String("broker", broker), zap.Error(err))
			continue
		}

		controller, err := conn.Controller()
		conn.Close()
		if err != nil {
			m.logger.Debug("failed to get controller from broker", zap.String("broker", broker), zap.Error(err))
			continue
		}

		m.controller = net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
		m.logger.Debug("found kafka controller", zap.String("controller", m.controller))
		return m.controller, nil
	}

	return "", fmt.Errorf("failed to find kafka controller from brokers: %v", m.brokers)
}

// dialBroker creates a connection to a broker.
func (m *TopicManager) dialBroker(ctx context.Context, addr string) (*kafka.Conn, error) {
	dialer := &kafka.Dialer{
		Timeout:   m.config.DialTimeout,
		DualStack: true,
	}
	return dialer.DialContext(ctx, "tcp", addr)
}

// dialController creates a connection to the controller broker.
func (m *TopicManager) dialController(ctx context.Context) (*kafka.Conn, error) {
	controller, err := m.getController(ctx)
	if err != nil {
		return nil, err
	}
	return m.dialBroker(ctx, controller)
}

// EnsureTopics ensures all configured topics exist.
func (m *TopicManager) EnsureTopics(ctx context.Context) error {
	if !m.config.AutoCreate {
		m.logger.Info("auto_create disabled, skipping topic creation")
		return nil
	}

	existingTopics, err := m.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("failed to list existing topics: %w", err)
	}

	existingSet := make(map[string]bool)
	for _, t := range existingTopics {
		existingSet[t] = true
	}

	var toCreate []kafka.TopicConfig
	for name, def := range m.config.TopicConfigs {
		if existingSet[name] {
			m.logger.Debug("topic already exists", zap.String("topic", name))
			continue
		}
		toCreate = append(toCreate, kafka.TopicConfig{
			Topic:             name,
			NumPartitions:     def.Partitions,
			ReplicationFactor: def.ReplicationFactor,
			ConfigEntries: []kafka.ConfigEntry{
				{ConfigName: "retention.ms", ConfigValue: strconv.FormatInt(def.RetentionMs, 10)},
				{ConfigName: "cleanup.policy", ConfigValue: def.CleanupPolicy},
			},
		})
	}

	if len(toCreate) == 0 {
		m.logger.Info("all topics already exist")
		return nil
	}

	conn, err := m.dialController(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(toCreate...)
	if err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}

	for _, tc := range toCreate {
		m.logger.Info("topic created",
			zap.String("topic", tc.Topic),
			zap.Int("partitions", tc.NumPartitions),
			zap.Int("replication_factor", tc.ReplicationFactor),
		)
	}

	return nil
}

// CreateTopic creates a single topic.
func (m *TopicManager) CreateTopic(ctx context.Context, name string, def TopicDefinition) error {
	conn, err := m.dialController(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer conn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     def.Partitions,
		ReplicationFactor: def.ReplicationFactor,
		ConfigEntries: []kafka.ConfigEntry{
			{ConfigName: "retention.ms", ConfigValue: strconv.FormatInt(def.RetentionMs, 10)},
			{ConfigName: "cleanup.policy", ConfigValue: def.CleanupPolicy},
		},
	}

	err = conn.CreateTopics(topicConfig)
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %w", name, err)
	}

	m.logger.Info("topic created",
		zap.String("topic", name),
		zap.Int("partitions", def.Partitions),
		zap.Int("replication_factor", def.ReplicationFactor),
		zap.Int64("retention_ms", def.RetentionMs),
	)

	return nil
}

// ListTopics lists all topics in the cluster.
func (m *TopicManager) ListTopics(ctx context.Context) ([]string, error) {
	conn, err := m.dialController(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions: %w", err)
	}

	topicSet := make(map[string]bool)
	for _, p := range partitions {
		topicSet[p.Topic] = true
	}

	topics := make([]string, 0, len(topicSet))
	for t := range topicSet {
		topics = append(topics, t)
	}

	return topics, nil
}

// DeleteTopic deletes a topic (use with caution).
func (m *TopicManager) DeleteTopic(ctx context.Context, name string) error {
	conn, err := m.dialController(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer conn.Close()

	err = conn.DeleteTopics(name)
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", name, err)
	}

	m.logger.Info("topic deleted", zap.String("topic", name))
	return nil
}

// GetTopicMetadata gets metadata for a specific topic.
func (m *TopicManager) GetTopicMetadata(ctx context.Context, name string) (*TopicMetadata, error) {
	conn, err := m.dialController(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions for topic %s: %w", name, err)
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("topic %s not found", name)
	}

	return &TopicMetadata{
		Name:       name,
		Partitions: len(partitions),
		// Note: ReplicationFactor requires additional API call
	}, nil
}

// TopicMetadata contains metadata about a topic.
type TopicMetadata struct {
	Name              string `json:"name"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor,omitempty"`
}

// Close closes the TopicManager.
func (m *TopicManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.controller = ""
	return nil
}

// TopicExists checks if a topic exists.
func (m *TopicManager) TopicExists(ctx context.Context, name string) (bool, error) {
	topics, err := m.ListTopics(ctx)
	if err != nil {
		return false, err
	}
	for _, t := range topics {
		if t == name {
			return true, nil
		}
	}
	return false, nil
}
