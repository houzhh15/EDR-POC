/**
 * Dashboard 数据类型定义
 * 仪表盘相关的所有数据接口和枚举类型
 */

/**
 * 时间范围类型
 * 用于筛选仪表盘数据的时间窗口
 */
export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d';

/**
 * 告警严重性等级
 */
export type AlertSeverity = 'critical' | 'high' | 'medium' | 'low';

/**
 * Severity别名（兼容性）
 */
export type Severity = AlertSeverity;

/**
 * 趋势方向
 */
export type TrendDirection = 'up' | 'down' | 'flat';

/**
 * 节点类型（用于攻击链）
 */
export type NodeType = 'process' | 'file' | 'network' | 'host' | 'user';

/**
 * 威胁类型（用于Top N分析）
 */
export type ThreatType = 'process' | 'ip' | 'domain';

/**
 * 告警统计数据
 */
export interface AlertStats {
  /** 严重告警数 */
  critical: number;
  /** 高危告警数 */
  high: number;
  /** 中危告警数 */
  medium: number;
  /** 低危告警数 */
  low: number;
  /** 总告警数 */
  total: number;
  /** 趋势数据（可选） */
  trend?: {
    direction: TrendDirection;
    value: number;
  };
}

/**
 * 终端统计数据
 */
export interface EndpointStats {
  /** 在线终端数 */
  online: number;
  /** 离线终端数 */
  offline: number;
  /** 有风险终端数 */
  at_risk: number;
  /** 总终端数 */
  total: number;
}

/**
 * 规则统计数据
 */
export interface RuleStats {
  /** 启用规则数 */
  enabled: number;
  /** 禁用规则数 */
  disabled: number;
  /** 总规则数 */
  total: number;
}

/**
 * 仪表盘统计数据
 * 包含告警、终端、规则、事件等关键指标
 */
export interface DashboardStats {
  /** 告警统计 */
  alerts: AlertStats;
  /** 终端统计 */
  endpoints: EndpointStats;
  /** 规则统计 */
  rules: RuleStats;
  /** 今日事件数 */
  events_today: number;
  /** 平均检测时间(分钟) */
  mttd_minutes: number;
  /** 平均响应时间(分钟) */
  mttr_minutes: number;
  /** 最后更新时间(ISO 8601) */
  last_updated: string;
}

/**
 * 告警趋势数据点
 * 用于绘制时间序列图表
 */
export interface AlertTrendPoint {
  /** 时间戳(ISO 8601) */
  timestamp: string;
  /** 严重告警数 */
  critical: number;
  /** 高危告警数 */
  high: number;
  /** 中危告警数 */
  medium: number;
  /** 低危告警数 */
  low: number;
  /** 总告警数 */
  total: number;
}

/**
 * Top N 列表项
 * 用于Top进程/IP/域名分析
 */
export interface TopNItem {
  /** 排名 */
  rank: number;
  /** 名称（进程名/IP地址/域名） */
  name: string;
  /** 命中次数 */
  count: number;
  /** 风险等级 */
  risk_level: AlertSeverity;
  /** 元数据（可选） */
  metadata?: Record<string, unknown>;
}

/**
 * MITRE ATT&CK 单元格数据
 * 用于热力图展示
 */
export interface MitreCell {
  /** 战术名称 */
  tactic: string;
  /** 技术ID（如T1059） */
  technique_id: string;
  /** 技术名称 */
  technique_name: string;
  /** 关联规则数 */
  rule_count: number;
  /** 命中次数 */
  hit_count: number;
}

/**
 * 攻击链节点
 */
export interface AttackChainNode {
  /** 节点ID */
  id: string;
  /** 节点类型 */
  type: NodeType;
  /** 节点标签 */
  label: string;
  /** 风险等级 */
  risk_level: AlertSeverity;
  /** 时间戳(ISO 8601) */
  timestamp: string;
  /** 元数据 */
  metadata: Record<string, unknown>;
}

/**
 * 攻击链边
 */
export interface AttackChainEdge {
  /** 源节点ID */
  source: string;
  /** 目标节点ID */
  target: string;
  /** 边标签（如spawned, accessed） */
  label: string;
}

/**
 * 攻击链
 * 包含节点和边的完整攻击链数据
 */
export interface AttackChain {
  /** 攻击链ID */
  id: string;
  /** 严重性 */
  severity: AlertSeverity;
  /** 创建时间(ISO 8601) */
  created_at: string;
  /** 节点列表 */
  nodes: AttackChainNode[];
  /** 边列表 */
  edges: AttackChainEdge[];
}

/**
 * API响应：仪表盘统计
 */
export interface DashboardStatsResponse {
  /** 统计数据 */
  data: DashboardStats;
}

/**
 * API响应：告警趋势
 */
export interface AlertTrendResponse {
  /** 时间范围 */
  range: TimeRange;
  /** 趋势数据 */
  data: AlertTrendPoint[];
}

/**
 * API响应：Top威胁
 */
export interface TopThreatsResponse {
  /** 威胁类型 */
  type: ThreatType;
  /** Top N数据 */
  data: TopNItem[];
}

/**
 * API响应：MITRE覆盖度
 */
export interface MitreCoverageResponse {
  /** 战术列表 */
  tactics: string[];
  /** 单元格数据 */
  data: MitreCell[];
}

/**
 * API响应：攻击链
 */
export interface AttackChainsResponse {
  /** 攻击链列表 */
  chains: AttackChain[];
}
