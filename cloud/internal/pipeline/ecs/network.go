// Package ecs provides network event mappers.
package ecs

// NetworkConnectMapper 网络连接事件映射器
type NetworkConnectMapper struct{}

// Map 映射网络连接事件
func (m *NetworkConnectMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"network"}
	ecs.Event.Type = []string{"connection", "start"}
	ecs.Event.Action = "network_connection"
	ecs.Event.Dataset = "network"

	if evt.Network != nil {
		// 源地址
		ecs.Source = &ECSSource{
			IP:   evt.Network.SourceIP,
			Port: evt.Network.SourcePort,
		}

		if evt.Network.BytesSent > 0 {
			ecs.Source.Bytes = evt.Network.BytesSent
		}

		// 目标地址
		ecs.Destination = &ECSDestination{
			IP:   evt.Network.DestinationIP,
			Port: evt.Network.DestinationPort,
		}

		if evt.Network.BytesReceived > 0 {
			ecs.Destination.Bytes = evt.Network.BytesReceived
		}

		// 网络元数据
		ecs.Network = &ECSNetwork{
			Protocol:  evt.Network.Protocol,
			Transport: evt.Network.Protocol, // tcp, udp
			Direction: evt.Network.Direction,
		}

		// 计算总字节数
		totalBytes := evt.Network.BytesSent + evt.Network.BytesReceived
		if totalBytes > 0 {
			ecs.Network.Bytes = totalBytes
		}
	}

	// 关联进程信息（如果有）
	if evt.Process != nil {
		ecs.Process = &ECSProcess{
			PID:        evt.Process.PID,
			Name:       evt.Process.Name,
			Executable: evt.Process.Executable,
		}
	}

	// 应用 GeoIP 丰富化数据
	if evt.Enrichment != nil && evt.Enrichment.GeoIP != nil {
		geo := evt.Enrichment.GeoIP

		// 为源地址添加地理信息（假设是出站连接）
		if ecs.Destination != nil && ecs.Destination.IP == evt.Network.DestinationIP {
			ecs.Destination.Geo = &ECSGeo{
				CountryIsoCode: geo.CountryCode,
				CountryName:    geo.CountryName,
				CityName:       geo.CityName,
			}

			if geo.Latitude != 0 || geo.Longitude != 0 {
				ecs.Destination.Geo.Location = &ECSGeoLocation{
					Lat: geo.Latitude,
					Lon: geo.Longitude,
				}
			}
		}
	}

	return nil
}

// NetworkDisconnectMapper 网络断开事件映射器
type NetworkDisconnectMapper struct{}

// Map 映射网络断开事件
func (m *NetworkDisconnectMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"network"}
	ecs.Event.Type = []string{"connection", "end"}
	ecs.Event.Action = "network_disconnection"
	ecs.Event.Dataset = "network"

	if evt.Network != nil {
		ecs.Source = &ECSSource{
			IP:   evt.Network.SourceIP,
			Port: evt.Network.SourcePort,
		}

		ecs.Destination = &ECSDestination{
			IP:   evt.Network.DestinationIP,
			Port: evt.Network.DestinationPort,
		}

		ecs.Network = &ECSNetwork{
			Protocol:  evt.Network.Protocol,
			Transport: evt.Network.Protocol,
			Direction: evt.Network.Direction,
			Bytes:     evt.Network.BytesSent + evt.Network.BytesReceived,
		}
	}

	return nil
}

// DNSQueryMapper DNS查询事件映射器
type DNSQueryMapper struct{}

// Map 映射DNS查询事件
func (m *DNSQueryMapper) Map(evt *Event, ecs *ECSEvent) error {
	ecs.Event.Category = []string{"network"}
	ecs.Event.Type = []string{"protocol"}
	ecs.Event.Action = "dns_query"
	ecs.Event.Dataset = "dns"

	if evt.Network != nil {
		ecs.Source = &ECSSource{
			IP:   evt.Network.SourceIP,
			Port: evt.Network.SourcePort,
		}

		ecs.Destination = &ECSDestination{
			IP:   evt.Network.DestinationIP,
			Port: evt.Network.DestinationPort,
		}

		ecs.Network = &ECSNetwork{
			Protocol:    "dns",
			Transport:   evt.Network.Protocol,
			Application: "dns",
		}
	}

	return nil
}
