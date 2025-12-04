// Package ecs provides event mappers for ECS conversion.
package ecs

// Event 内部事件结构（管线处理中间态）
type Event struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	TenantID  string `json:"tenant_id"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp in nanoseconds
	EventType string `json:"event_type"`

	Process  *ProcessInfo  `json:"process,omitempty"`
	File     *FileInfo     `json:"file,omitempty"`
	Network  *NetworkInfo  `json:"network,omitempty"`
	Registry *RegistryInfo `json:"registry,omitempty"`
	User     *UserInfo     `json:"user,omitempty"`

	Enrichment *EnrichmentData `json:"enrichment,omitempty"`
	RawData    []byte          `json:"-"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID         int32    `json:"pid"`
	PPID        int32    `json:"ppid"`
	Name        string   `json:"name"`
	Executable  string   `json:"executable"`
	CommandLine string   `json:"command_line"`
	Args        []string `json:"args,omitempty"`
	WorkingDir  string   `json:"working_dir,omitempty"`
	Hash        *Hash    `json:"hash,omitempty"`
	User        string   `json:"user,omitempty"`
	StartTime   int64    `json:"start_time,omitempty"`
	ExitCode    int      `json:"exit_code,omitempty"`
}

// FileInfo 文件信息
type FileInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Directory string `json:"directory,omitempty"`
	Size      int64  `json:"size"`
	Hash      *Hash  `json:"hash,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Created   int64  `json:"created,omitempty"`
	Modified  int64  `json:"modified,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	SourceIP        string `json:"source_ip"`
	SourcePort      int    `json:"source_port"`
	DestinationIP   string `json:"destination_ip"`
	DestinationPort int    `json:"destination_port"`
	Protocol        string `json:"protocol"`
	Direction       string `json:"direction"`
	BytesSent       int64  `json:"bytes_sent,omitempty"`
	BytesReceived   int64  `json:"bytes_received,omitempty"`
}

// RegistryInfo 注册表信息 (Windows)
type RegistryInfo struct {
	Hive      string `json:"hive"`
	Key       string `json:"key"`
	ValueName string `json:"value_name,omitempty"`
	ValueType string `json:"value_type,omitempty"`
	ValueData string `json:"value_data,omitempty"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
}

// Hash 哈希值
type Hash struct {
	MD5    string `json:"md5,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// EnrichmentData 丰富化数据
type EnrichmentData struct {
	GeoIP *GeoIPInfo `json:"geoip,omitempty"`
	Asset *AssetInfo `json:"asset,omitempty"`
	Agent *AgentInfo `json:"agent,omitempty"`
}

// GeoIPInfo GeoIP信息
type GeoIPInfo struct {
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	CityName    string  `json:"city_name,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	ASN         int     `json:"asn,omitempty"`
	ASOrg       string  `json:"as_org,omitempty"`
}

// AssetInfo 资产信息
type AssetInfo struct {
	Hostname   string            `json:"hostname"`
	OSFamily   string            `json:"os_family"`
	OSVersion  string            `json:"os_version"`
	Department string            `json:"department,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// AgentInfo Agent信息
type AgentInfo struct {
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
}

// EventMapper 事件映射器接口
type EventMapper interface {
	// Map 将内部事件映射到 ECS 格式
	Map(evt *Event, ecs *ECSEvent) error
}

// convertHash 将内部 Hash 转换为 ECS Hash
func convertHash(h *Hash) *ECSHash {
	if h == nil {
		return nil
	}
	return &ECSHash{
		MD5:    h.MD5,
		SHA1:   h.SHA1,
		SHA256: h.SHA256,
	}
}
