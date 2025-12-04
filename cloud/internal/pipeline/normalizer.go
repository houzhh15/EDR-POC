// Package pipeline provides the event normalizer for ECS conversion.
package pipeline

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/houzhh15/EDR-POC/cloud/internal/pipeline/ecs"
)

// Normalizer 事件标准化器接口
type Normalizer interface {
	Normalize(ctx context.Context, event *ecs.Event) (*ecs.ECSEvent, error)
	SupportedTypes() []string
}

// ECSNormalizer ECS标准化器实现
type ECSNormalizer struct {
	mappers map[string]ecs.EventMapper
	logger  *zap.Logger
}

// NewECSNormalizer 创建 ECS 标准化器
func NewECSNormalizer(logger *zap.Logger) *ECSNormalizer {
	if logger == nil {
		logger = zap.NewNop()
	}

	n := &ECSNormalizer{
		mappers: make(map[string]ecs.EventMapper),
		logger:  logger,
	}

	// 注册所有支持的事件类型映射器
	n.mappers["process_create"] = &ecs.ProcessCreateMapper{}
	n.mappers["process_terminate"] = &ecs.ProcessTerminateMapper{}
	n.mappers["file_create"] = &ecs.FileCreateMapper{}
	n.mappers["file_modify"] = &ecs.FileModifyMapper{}
	n.mappers["file_delete"] = &ecs.FileDeleteMapper{}
	n.mappers["network_connect"] = &ecs.NetworkConnectMapper{}
	n.mappers["network_disconnect"] = &ecs.NetworkDisconnectMapper{}
	n.mappers["dns_query"] = &ecs.DNSQueryMapper{}

	logger.Info("ECS normalizer initialized", zap.Int("mapper_count", len(n.mappers)))

	return n
}

// Normalize 将内部事件转换为 ECS 格式
func (n *ECSNormalizer) Normalize(ctx context.Context, evt *ecs.Event) (*ecs.ECSEvent, error) {
	if evt == nil {
		return nil, fmt.Errorf("event is nil")
	}

	mapper, ok := n.mappers[evt.EventType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedEventType, evt.EventType)
	}

	// 创建基础 ECS 事件
	ecsEvent := &ecs.ECSEvent{
		Timestamp: time.Unix(0, evt.Timestamp),
		ECS:       ecs.ECSMeta{Version: "8.11.0"},
		Event: ecs.ECSEventMeta{
			ID:       evt.ID,
			Kind:     "event",
			Module:   "edr",
			Provider: "edr-agent",
			Created:  time.Now(),
			Ingested: time.Now(),
		},
		Agent: ecs.ECSAgent{
			ID:   evt.AgentID,
			Type: "edr",
		},
	}

	// 调用映射器填充具体字段
	if err := mapper.Map(evt, ecsEvent); err != nil {
		return nil, fmt.Errorf("mapper error: %w", err)
	}

	// 应用丰富化数据
	n.applyEnrichment(evt, ecsEvent)

	return ecsEvent, nil
}

// SupportedTypes 返回所有支持的事件类型
func (n *ECSNormalizer) SupportedTypes() []string {
	types := make([]string, 0, len(n.mappers))
	for k := range n.mappers {
		types = append(types, k)
	}
	return types
}

// RegisterMapper 注册自定义映射器
func (n *ECSNormalizer) RegisterMapper(eventType string, mapper ecs.EventMapper) {
	n.mappers[eventType] = mapper
	n.logger.Info("registered custom mapper", zap.String("event_type", eventType))
}

// applyEnrichment 应用丰富化数据到 ECS 事件
func (n *ECSNormalizer) applyEnrichment(evt *ecs.Event, ecsEvent *ecs.ECSEvent) {
	if evt.Enrichment == nil {
		return
	}

	// 应用 GeoIP 数据
	if evt.Enrichment.GeoIP != nil {
		geo := &ecs.ECSGeo{
			CountryIsoCode: evt.Enrichment.GeoIP.CountryCode,
			CountryName:    evt.Enrichment.GeoIP.CountryName,
			CityName:       evt.Enrichment.GeoIP.CityName,
		}

		if evt.Enrichment.GeoIP.Latitude != 0 || evt.Enrichment.GeoIP.Longitude != 0 {
			geo.Location = &ecs.ECSGeoLocation{
				Lat: evt.Enrichment.GeoIP.Latitude,
				Lon: evt.Enrichment.GeoIP.Longitude,
			}
		}

		// 将 GeoIP 应用到目标地址（出站连接场景）
		if ecsEvent.Destination != nil {
			ecsEvent.Destination.Geo = geo
		} else if ecsEvent.Source != nil {
			ecsEvent.Source.Geo = geo
		}
	}

	// 应用资产数据
	if evt.Enrichment.Asset != nil {
		ecsEvent.Host = ecs.ECSHost{
			Hostname: evt.Enrichment.Asset.Hostname,
			OS: &ecs.ECSOS{
				Family:  evt.Enrichment.Asset.OSFamily,
				Version: evt.Enrichment.Asset.OSVersion,
			},
		}

		// 将资产标签添加到 Labels
		if len(evt.Enrichment.Asset.Labels) > 0 {
			if ecsEvent.Labels == nil {
				ecsEvent.Labels = make(map[string]string)
			}
			for k, v := range evt.Enrichment.Asset.Labels {
				ecsEvent.Labels[k] = v
			}
		}

		// 将资产 Tags 添加到 ECS Tags
		if len(evt.Enrichment.Asset.Tags) > 0 {
			ecsEvent.Tags = append(ecsEvent.Tags, evt.Enrichment.Asset.Tags...)
		}

		// 添加部门信息到 Labels
		if evt.Enrichment.Asset.Department != "" {
			if ecsEvent.Labels == nil {
				ecsEvent.Labels = make(map[string]string)
			}
			ecsEvent.Labels["department"] = evt.Enrichment.Asset.Department
		}
	}

	// 应用 Agent 数据
	if evt.Enrichment.Agent != nil {
		ecsEvent.Agent.Version = evt.Enrichment.Agent.Version
		ecsEvent.Agent.Name = evt.Enrichment.Agent.Hostname
	}
}
