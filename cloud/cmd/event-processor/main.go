// Package main 是 Event Processor 服务的入口点
//
// Event Processor 负责：
//   - 从 Kafka 消费事件
//   - 事件解析和标准化
//   - 事件存储（写入 OpenSearch）
//   - 事件关联和富化
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/enricher"
	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/writer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// 版本信息
var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var (
	configFile  = flag.String("config", "configs/pipeline.yaml", "配置文件路径")
	metricsAddr = flag.String("metrics", ":9091", "指标服务地址")
)

func main() {
	flag.Parse()

	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Event Processor starting",
		zap.String("version", Version),
		zap.String("commit", GitCommit),
	)

	// 加载配置
	cfg, err := loadConfig(*configFile)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 创建指标
	metrics := pipeline.NewPipelineMetrics("edr")
	metrics.MustRegister(prometheus.DefaultRegisterer)

	// 创建富化器
	enrichers, err := createEnrichers(cfg, logger)
	if err != nil {
		logger.Fatal("创建富化器失败", zap.Error(err))
	}

	// 创建标准化器
	normalizer := pipeline.NewECSNormalizer(logger)

	// 创建输出写入器
	writers, dlqWriter, err := createWriters(cfg, logger)
	if err != nil {
		logger.Fatal("创建写入器失败", zap.Error(err))
	}

	// 创建管线
	p, err := pipeline.NewPipeline(cfg, enrichers, normalizer, writers, dlqWriter, metrics)
	if err != nil {
		logger.Fatal("创建管线失败", zap.Error(err))
	}

	// 启动指标服务
	go startMetricsServer(*metricsAddr, logger)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动管线
	if err := p.Start(ctx); err != nil {
		logger.Fatal("启动管线失败", zap.Error(err))
	}

	logger.Info("事件处理管线服务已启动")

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))

	// 优雅停止
	if err := p.Stop(); err != nil {
		logger.Error("停止管线失败", zap.Error(err))
	}

	logger.Info("Event Processor stopped")
}

// loadConfig 从文件加载配置
func loadConfig(path string) (*pipeline.PipelineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件: %w", err)
	}

	var cfg pipeline.PipelineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件: %w", err)
	}

	// 设置默认值
	if cfg.Processing.BatchSize <= 0 {
		cfg.Processing.BatchSize = 1000
	}
	if cfg.Processing.BatchTimeout <= 0 {
		cfg.Processing.BatchTimeout = 100 * time.Millisecond
	}
	if cfg.Processing.WorkerCount <= 0 {
		cfg.Processing.WorkerCount = 4
	}

	return &cfg, nil
}

// createEnrichers 创建富化器链
func createEnrichers(cfg *pipeline.PipelineConfig, logger *zap.Logger) ([]enricher.Enricher, error) {
	var enrichers []enricher.Enricher

	// GeoIP 富化器
	if cfg.Enrichment.GeoIP.Enabled {
		geoipEnricher, err := enricher.NewGeoIPEnricher(&enricher.GeoIPEnricherConfig{
			Enabled:      true,
			DatabasePath: cfg.Enrichment.GeoIP.DatabasePath,
		}, logger)
		if err != nil {
			logger.Warn("创建GeoIP富化器失败，跳过", zap.Error(err))
		} else {
			enrichers = append(enrichers, geoipEnricher)
			logger.Info("GeoIP富化器已启用")
		}
	}

	// Asset 富化器
	if cfg.Enrichment.Asset.Enabled {
		assetEnricher, _ := enricher.NewAssetEnricher(&enricher.AssetEnricherConfig{
			Enabled:  true,
			CacheTTL: cfg.Enrichment.Asset.CacheTTL,
		}, logger)
		enrichers = append(enrichers, assetEnricher)
		logger.Info("Asset富化器已启用")
	}

	// Agent 富化器
	if cfg.Enrichment.Agent.Enabled {
		agentEnricher, _ := enricher.NewAgentEnricher(&enricher.AgentEnricherConfig{
			Enabled:  true,
			CacheTTL: cfg.Enrichment.Agent.CacheTTL,
		}, logger)
		enrichers = append(enrichers, agentEnricher)
		logger.Info("Agent富化器已启用")
	}

	return enrichers, nil
}

// createWriters 创建输出写入器
func createWriters(cfg *pipeline.PipelineConfig, logger *zap.Logger) ([]writer.Writer, writer.Writer, error) {
	var writers []writer.Writer

	// Kafka 输出
	if cfg.Output.Kafka.Enabled {
		kafkaWriter, err := writer.NewKafkaWriter(&writer.KafkaWriterConfig{
			Brokers:      cfg.Output.Kafka.Brokers,
			Topic:        cfg.Output.Kafka.Topic,
			BatchSize:    cfg.Output.Kafka.BatchSize,
			BatchTimeout: cfg.Output.Kafka.BatchTimeout,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建Kafka写入器: %w", err)
		}
		writers = append(writers, kafkaWriter)
		logger.Info("Kafka写入器已启用", zap.String("topic", cfg.Output.Kafka.Topic))
	}

	// OpenSearch 输出
	if cfg.Output.OpenSearch.Enabled {
		osWriter, err := writer.NewOpenSearchWriter(&writer.OpenSearchConfig{
			Addresses:     cfg.Output.OpenSearch.Addresses,
			Index:         cfg.Output.OpenSearch.IndexPrefix,
			IndexRotation: "daily",
			BatchSize:     cfg.Output.OpenSearch.BulkSize,
			FlushInterval: cfg.Output.OpenSearch.FlushInterval,
			Username:      cfg.Output.OpenSearch.Username,
			Password:      cfg.Output.OpenSearch.Password,
			TLSEnabled:    cfg.Output.OpenSearch.TLSEnabled,
			TLSInsecure:   cfg.Output.OpenSearch.TLSSkipVerify,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建OpenSearch写入器: %w", err)
		}
		writers = append(writers, osWriter)
		logger.Info("OpenSearch写入器已启用", zap.Strings("addresses", cfg.Output.OpenSearch.Addresses))
	}

	// DLQ 写入器
	var dlqWriter writer.Writer
	if cfg.ErrorHandling.DLQTopic != "" {
		var err error
		dlqWriter, err = writer.NewKafkaWriter(&writer.KafkaWriterConfig{
			Brokers:   cfg.Input.Kafka.Brokers, // 复用输入配置的 brokers
			Topic:     cfg.ErrorHandling.DLQTopic,
			BatchSize: 100,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("创建DLQ写入器: %w", err)
		}
		logger.Info("DLQ写入器已启用", zap.String("topic", cfg.ErrorHandling.DLQTopic))
	}

	return writers, dlqWriter, nil
}

// startMetricsServer 启动指标服务
func startMetricsServer(addr string, logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("指标服务启动", zap.String("addr", addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("指标服务错误", zap.Error(err))
	}
}

// healthHandler 健康检查处理器
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readyHandler 就绪检查处理器
func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
