// Package enricher provides GeoIP enrichment functionality.
package enricher

import (
	"context"
	"net"

	"github.com/oschwald/geoip2-golang"
	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// GeoIPEnricher GeoIP丰富化器
type GeoIPEnricher struct {
	db      *geoip2.Reader
	enabled bool
	logger  *zap.Logger
}

// GeoIPEnricherConfig GeoIP丰富化器配置
type GeoIPEnricherConfig struct {
	Enabled      bool   `yaml:"enabled"`
	DatabasePath string `yaml:"database_path"`
}

// NewGeoIPEnricher 创建GeoIP丰富化器
func NewGeoIPEnricher(cfg *GeoIPEnricherConfig, logger *zap.Logger) (*GeoIPEnricher, error) {
	if !cfg.Enabled {
		return &GeoIPEnricher{enabled: false, logger: logger}, nil
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	db, err := geoip2.Open(cfg.DatabasePath)
	if err != nil {
		logger.Warn("failed to open GeoIP database, enricher disabled",
			zap.String("path", cfg.DatabasePath), zap.Error(err))
		return &GeoIPEnricher{enabled: false, logger: logger}, nil
	}

	logger.Info("GeoIP enricher initialized", zap.String("database_path", cfg.DatabasePath))
	return &GeoIPEnricher{db: db, enabled: true, logger: logger}, nil
}

// Name 返回丰富化器名称
func (e *GeoIPEnricher) Name() string { return "geoip" }

// Enabled 返回是否启用
func (e *GeoIPEnricher) Enabled() bool { return e.enabled }

// Enrich 丰富化事件
func (e *GeoIPEnricher) Enrich(ctx context.Context, evt *ecs.Event) error {
	if !e.enabled || e.db == nil || evt.Network == nil {
		return nil
	}

	ipStr := evt.Network.DestinationIP
	if ipStr == "" {
		ipStr = evt.Network.SourceIP
	}
	if ipStr == "" {
		return nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || isPrivateIP(ip) {
		return nil
	}

	record, err := e.db.City(ip)
	if err != nil {
		e.logger.Debug("GeoIP lookup failed", zap.String("ip", ipStr), zap.Error(err))
		return nil
	}

	if evt.Enrichment == nil {
		evt.Enrichment = &ecs.EnrichmentData{}
	}

	evt.Enrichment.GeoIP = &ecs.GeoIPInfo{
		CountryCode: record.Country.IsoCode,
		CountryName: record.Country.Names["en"],
		CityName:    record.City.Names["en"],
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
	}

	return nil
}

// Close 关闭丰富化器
func (e *GeoIPEnricher) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// isPrivateIP 判断是否为私有 IP
func isPrivateIP(ip net.IP) bool {
	privateBlocks := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "169.254.0.0/16", "::1/128", "fe80::/10", "fc00::/7",
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
