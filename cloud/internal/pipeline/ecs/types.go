// Package ecs provides ECS (Elastic Common Schema) data structures and mappers.
package ecs

import (
	"time"
)

// ECSEvent ECS标准事件结构（ECS 8.11.0）
type ECSEvent struct {
	Timestamp   time.Time         `json:"@timestamp"`
	ECS         ECSMeta           `json:"ecs"`
	Event       ECSEventMeta      `json:"event"`
	Agent       ECSAgent          `json:"agent"`
	Host        ECSHost           `json:"host"`
	Process     *ECSProcess       `json:"process,omitempty"`
	File        *ECSFile          `json:"file,omitempty"`
	Source      *ECSSource        `json:"source,omitempty"`
	Destination *ECSDestination   `json:"destination,omitempty"`
	User        *ECSUser          `json:"user,omitempty"`
	Network     *ECSNetwork       `json:"network,omitempty"`
	Registry    *ECSRegistry      `json:"registry,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// ECSMeta ECS版本信息
type ECSMeta struct {
	Version string `json:"version"` // "8.11.0"
}

// ECSEventMeta 事件元数据
type ECSEventMeta struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`               // event, alert, metric, state
	Category  []string  `json:"category"`           // process, file, network, authentication, etc.
	Type      []string  `json:"type"`               // start, end, creation, deletion, etc.
	Action    string    `json:"action,omitempty"`   // process_created, file_written, etc.
	Outcome   string    `json:"outcome,omitempty"`  // success, failure, unknown
	Module    string    `json:"module,omitempty"`   // edr
	Dataset   string    `json:"dataset,omitempty"`  // process, file, network
	Provider  string    `json:"provider,omitempty"` // edr-agent
	Severity  int       `json:"severity,omitempty"` // 0-100
	RiskScore float64   `json:"risk_score,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Ingested  time.Time `json:"ingested,omitempty"`
	Start     time.Time `json:"start,omitempty"`
	End       time.Time `json:"end,omitempty"`
	Duration  int64     `json:"duration,omitempty"` // nanoseconds
}

// ECSAgent Agent信息
type ECSAgent struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"` // edr
	Version     string `json:"version,omitempty"`
	EphemeralID string `json:"ephemeral_id,omitempty"`
}

// ECSHost 主机信息
type ECSHost struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	Hostname     string   `json:"hostname,omitempty"`
	Architecture string   `json:"architecture,omitempty"` // x86_64, arm64
	IP           []string `json:"ip,omitempty"`
	MAC          []string `json:"mac,omitempty"`
	OS           *ECSOS   `json:"os,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	Type         string   `json:"type,omitempty"` // server, workstation
	Uptime       int64    `json:"uptime,omitempty"`
}

// ECSOS 操作系统信息
type ECSOS struct {
	Family   string `json:"family,omitempty"` // windows, linux, macos
	Kernel   string `json:"kernel,omitempty"`
	Name     string `json:"name,omitempty"`     // Windows 11, Ubuntu 22.04
	Platform string `json:"platform,omitempty"` // windows, debian, darwin
	Type     string `json:"type,omitempty"`     // windows, linux, macos
	Version  string `json:"version,omitempty"`
}

// ECSProcess 进程信息
type ECSProcess struct {
	PID           int32             `json:"pid"`
	Name          string            `json:"name,omitempty"`
	Executable    string            `json:"executable,omitempty"`
	CommandLine   string            `json:"command_line,omitempty"`
	Args          []string          `json:"args,omitempty"`
	ArgsCount     int               `json:"args_count,omitempty"`
	WorkingDir    string            `json:"working_directory,omitempty"`
	Hash          *ECSHash          `json:"hash,omitempty"`
	Parent        *ECSProcessParent `json:"parent,omitempty"`
	User          *ECSUser          `json:"user,omitempty"`
	Start         time.Time         `json:"start,omitempty"`
	End           time.Time         `json:"end,omitempty"`
	ExitCode      int               `json:"exit_code,omitempty"`
	EntityID      string            `json:"entity_id,omitempty"`
	CodeSignature *ECSCodeSignature `json:"code_signature,omitempty"`
	ThreadID      int32             `json:"thread.id,omitempty"`
}

// ECSProcessParent 父进程信息
type ECSProcessParent struct {
	PID         int32    `json:"pid"`
	Name        string   `json:"name,omitempty"`
	Executable  string   `json:"executable,omitempty"`
	CommandLine string   `json:"command_line,omitempty"`
	Args        []string `json:"args,omitempty"`
	EntityID    string   `json:"entity_id,omitempty"`
}

// ECSHash 哈希值
type ECSHash struct {
	MD5    string `json:"md5,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	SHA512 string `json:"sha512,omitempty"`
}

// ECSCodeSignature 代码签名
type ECSCodeSignature struct {
	Exists      bool   `json:"exists,omitempty"`
	SubjectName string `json:"subject_name,omitempty"`
	Valid       bool   `json:"valid,omitempty"`
	Trusted     bool   `json:"trusted,omitempty"`
	Status      string `json:"status,omitempty"`
	SigningID   string `json:"signing_id,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
}

// ECSFile 文件信息
type ECSFile struct {
	Path        string    `json:"path,omitempty"`
	Name        string    `json:"name,omitempty"`
	Extension   string    `json:"extension,omitempty"`
	Directory   string    `json:"directory,omitempty"`
	Size        int64     `json:"size,omitempty"`
	MimeType    string    `json:"mime_type,omitempty"`
	Type        string    `json:"type,omitempty"` // file, dir, symlink
	Mode        string    `json:"mode,omitempty"` // 0644
	UID         string    `json:"uid,omitempty"`
	GID         string    `json:"gid,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	Group       string    `json:"group,omitempty"`
	Hash        *ECSHash  `json:"hash,omitempty"`
	Device      string    `json:"device,omitempty"`
	Inode       string    `json:"inode,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Accessed    time.Time `json:"accessed,omitempty"`
	Mtime       time.Time `json:"mtime,omitempty"`
	Ctime       time.Time `json:"ctime,omitempty"`
	Attributes  []string  `json:"attributes,omitempty"`
	TargetPath  string    `json:"target_path,omitempty"`  // for symlinks
	DriveLetter string    `json:"drive_letter,omitempty"` // Windows
}

// ECSSource 源地址
type ECSSource struct {
	IP      string   `json:"ip,omitempty"`
	Port    int      `json:"port,omitempty"`
	Address string   `json:"address,omitempty"`
	Domain  string   `json:"domain,omitempty"`
	Bytes   int64    `json:"bytes,omitempty"`
	Packets int64    `json:"packets,omitempty"`
	Geo     *ECSGeo  `json:"geo,omitempty"`
	User    *ECSUser `json:"user,omitempty"`
}

// ECSDestination 目标地址
type ECSDestination struct {
	IP      string   `json:"ip,omitempty"`
	Port    int      `json:"port,omitempty"`
	Address string   `json:"address,omitempty"`
	Domain  string   `json:"domain,omitempty"`
	Bytes   int64    `json:"bytes,omitempty"`
	Packets int64    `json:"packets,omitempty"`
	Geo     *ECSGeo  `json:"geo,omitempty"`
	User    *ECSUser `json:"user,omitempty"`
}

// ECSGeo 地理位置信息
type ECSGeo struct {
	CityName       string          `json:"city_name,omitempty"`
	ContinentCode  string          `json:"continent_code,omitempty"`
	ContinentName  string          `json:"continent_name,omitempty"`
	CountryIsoCode string          `json:"country_iso_code,omitempty"`
	CountryName    string          `json:"country_name,omitempty"`
	Location       *ECSGeoLocation `json:"location,omitempty"`
	Name           string          `json:"name,omitempty"`
	PostalCode     string          `json:"postal_code,omitempty"`
	RegionIsoCode  string          `json:"region_iso_code,omitempty"`
	RegionName     string          `json:"region_name,omitempty"`
	Timezone       string          `json:"timezone,omitempty"`
}

// ECSGeoLocation 地理坐标
type ECSGeoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// ECSUser 用户信息
type ECSUser struct {
	ID     string    `json:"id,omitempty"`
	Name   string    `json:"name,omitempty"`
	Domain string    `json:"domain,omitempty"`
	Email  string    `json:"email,omitempty"`
	Roles  []string  `json:"roles,omitempty"`
	Group  *ECSGroup `json:"group,omitempty"`
}

// ECSGroup 用户组信息
type ECSGroup struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// ECSNetwork 网络信息
type ECSNetwork struct {
	Application string `json:"application,omitempty"` // http, dns, tls
	Protocol    string `json:"protocol,omitempty"`    // tcp, udp
	Transport   string `json:"transport,omitempty"`   // tcp, udp
	Type        string `json:"type,omitempty"`        // ipv4, ipv6
	Direction   string `json:"direction,omitempty"`   // inbound, outbound, internal
	Bytes       int64  `json:"bytes,omitempty"`
	Packets     int64  `json:"packets,omitempty"`
	CommunityID string `json:"community_id,omitempty"` // 1:hash
}

// ECSRegistry Windows注册表信息
type ECSRegistry struct {
	Hive  string           `json:"hive,omitempty"` // HKEY_LOCAL_MACHINE
	Key   string           `json:"key,omitempty"`
	Path  string           `json:"path,omitempty"`
	Value string           `json:"value,omitempty"`
	Data  *ECSRegistryData `json:"data,omitempty"`
}

// ECSRegistryData 注册表数据
type ECSRegistryData struct {
	Bytes   []byte   `json:"bytes,omitempty"`
	Strings []string `json:"strings,omitempty"`
	Type    string   `json:"type,omitempty"` // REG_SZ, REG_DWORD, etc.
}

// NewECSEvent 创建新的 ECS 事件
func NewECSEvent() *ECSEvent {
	return &ECSEvent{
		Timestamp: time.Now(),
		ECS:       ECSMeta{Version: "8.11.0"},
		Event: ECSEventMeta{
			Kind:     "event",
			Module:   "edr",
			Provider: "edr-agent",
		},
	}
}
