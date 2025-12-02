/**
 * Dashboard API Mock数据
 * 用于开发和测试环境
 */
import dayjs from 'dayjs';
import type {
  DashboardStats,
  AlertTrendPoint,
  TopNItem,
  MitreCell,
  AttackChain,
  TimeRange,
  ThreatType,
} from '../../types/dashboard';

/**
 * 模拟网络延迟
 */
const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

/**
 * Mock: 获取仪表盘统计数据
 */
export const getDashboardStats = async (): Promise<DashboardStats> => {
  await delay(300);
  
  return {
    alerts: {
      critical: 5,
      high: 12,
      medium: 45,
      low: 128,
      total: 190,
      trend: { direction: 'up', value: 12 },
    },
    endpoints: {
      online: 856,
      offline: 23,
      at_risk: 15,
      total: 894,
    },
    rules: {
      enabled: 156,
      disabled: 12,
      total: 168,
    },
    events_today: 12456,
    mttd_minutes: 8.5,
    mttr_minutes: 45.2,
    last_updated: new Date().toISOString(),
  };
};

/**
 * Mock: 获取告警趋势数据
 */
export const getAlertTrend = async (range: TimeRange): Promise<AlertTrendPoint[]> => {
  await delay(400);

  const points: number = range === '24h' ? 24 : range === '7d' ? 7 : 30;
  const data: AlertTrendPoint[] = [];

  for (let i = points - 1; i >= 0; i--) {
    const timestamp =
      range === '24h'
        ? dayjs().subtract(i, 'hour').toISOString()
        : dayjs().subtract(i, 'day').toISOString();

    data.push({
      timestamp,
      critical: Math.floor(Math.random() * 10) + 1,
      high: Math.floor(Math.random() * 20) + 5,
      medium: Math.floor(Math.random() * 50) + 20,
      low: Math.floor(Math.random() * 150) + 50,
      total: 0, // 将在下面计算
    });

    // 计算total
    const lastItem = data[data.length - 1];
    lastItem.total =
      lastItem.critical + lastItem.high + lastItem.medium + lastItem.low;
  }

  return data;
};

/**
 * Mock: 获取Top威胁数据
 */
export const getTopThreats = async (
  type: ThreatType,
  limit: number = 10
): Promise<TopNItem[]> => {
  await delay(350);

  const mockData: Record<ThreatType, string[]> = {
    process: [
      'powershell.exe',
      'cmd.exe',
      'rundll32.exe',
      'regsvr32.exe',
      'wscript.exe',
      'cscript.exe',
      'mshta.exe',
      'certutil.exe',
      'bitsadmin.exe',
      'schtasks.exe',
    ],
    ip: [
      '192.168.1.100',
      '10.0.0.25',
      '172.16.0.88',
      '203.0.113.45',
      '198.51.100.12',
      '192.0.2.156',
      '10.10.10.99',
      '172.31.255.1',
      '8.8.8.8',
      '1.1.1.1',
    ],
    domain: [
      'malicious-domain.com',
      'suspicious-site.net',
      'phishing-page.org',
      'evil-corp.biz',
      'bad-actors.info',
      'threat-source.xyz',
      'malware-host.cn',
      'exploit-kit.ru',
      'c2-server.su',
      'botnet-node.to',
    ],
  };

  const severities: ('critical' | 'high' | 'medium' | 'low')[] = [
    'critical',
    'high',
    'medium',
    'low',
  ];

  return mockData[type].slice(0, limit).map((name, index) => ({
    rank: index + 1,
    name,
    count: Math.floor(Math.random() * (200 - index * 15)) + 50,
    risk_level: severities[Math.floor(Math.random() * severities.length)],
    metadata: {
      last_seen: dayjs()
        .subtract(Math.floor(Math.random() * 24), 'hour')
        .toISOString(),
    },
  }));
};

/**
 * Mock: 获取MITRE ATT&CK覆盖度数据
 */
export const getMitreCoverage = async (): Promise<MitreCell[]> => {
  await delay(500);

  const tactics = [
    'Reconnaissance',
    'Resource Development',
    'Initial Access',
    'Execution',
    'Persistence',
    'Privilege Escalation',
    'Defense Evasion',
    'Credential Access',
    'Discovery',
    'Lateral Movement',
    'Collection',
    'Command and Control',
    'Exfiltration',
    'Impact',
  ];

  const techniques: Array<{ id: string; name: string }> = [
    { id: 'T1059', name: 'Command and Scripting Interpreter' },
    { id: 'T1055', name: 'Process Injection' },
    { id: 'T1003', name: 'OS Credential Dumping' },
    { id: 'T1082', name: 'System Information Discovery' },
    { id: 'T1071', name: 'Application Layer Protocol' },
    { id: 'T1027', name: 'Obfuscated Files or Information' },
    { id: 'T1547', name: 'Boot or Logon Autostart Execution' },
    { id: 'T1078', name: 'Valid Accounts' },
    { id: 'T1021', name: 'Remote Services' },
    { id: 'T1486', name: 'Data Encrypted for Impact' },
    { id: 'T1562', name: 'Impair Defenses' },
    { id: 'T1105', name: 'Ingress Tool Transfer' },
    { id: 'T1569', name: 'System Services' },
    { id: 'T1053', name: 'Scheduled Task/Job' },
    { id: 'T1070', name: 'Indicator Removal' },
  ];

  const data: MitreCell[] = [];

  // 生成部分覆盖的技术（不是所有战术都覆盖所有技术）
  for (const tactic of tactics) {
    // 每个战术随机选择3-6个技术
    const numTechniques = Math.floor(Math.random() * 4) + 3;
    const selectedTechniques = techniques
      .sort(() => Math.random() - 0.5)
      .slice(0, numTechniques);

    for (const technique of selectedTechniques) {
      data.push({
        tactic,
        technique_id: technique.id,
        technique_name: technique.name,
        rule_count: Math.floor(Math.random() * 10) + 1,
        hit_count: Math.floor(Math.random() * 500),
      });
    }
  }

  return data;
};

/**
 * Mock: 获取攻击链数据
 */
export const getAttackChains = async (limit: number = 5): Promise<AttackChain[]> => {
  await delay(450);

  const chains: AttackChain[] = [];
  const severities: ('critical' | 'high' | 'medium' | 'low')[] = [
    'critical',
    'high',
    'medium',
  ];

  for (let i = 0; i < Math.min(limit, 5); i++) {
    const numNodes = Math.floor(Math.random() * 6) + 5;
    const nodes = [];
    const edges = [];

    for (let j = 0; j < numNodes; j++) {
      const nodeTypes: ('process' | 'file' | 'network' | 'host' | 'user')[] = [
        'process',
        'file',
        'network',
        'host',
        'user',
      ];
      const type = nodeTypes[Math.floor(Math.random() * nodeTypes.length)];

      nodes.push({
        id: `node-${i}-${j}`,
        type,
        label:
          type === 'process'
            ? ['cmd.exe', 'powershell.exe', 'explorer.exe'][
                Math.floor(Math.random() * 3)
              ]
            : type === 'file'
            ? `file-${j}.exe`
            : type === 'network'
            ? `192.168.1.${j + 10}`
            : type === 'host'
            ? `host-${j}`
            : `user-${j}`,
        risk_level: severities[Math.floor(Math.random() * severities.length)],
        timestamp: dayjs()
          .subtract(numNodes - j, 'minute')
          .toISOString(),
        metadata: {
          pid: type === 'process' ? Math.floor(Math.random() * 10000) : undefined,
        },
      });

      // 添加边（除了第一个节点）
      if (j > 0) {
        edges.push({
          source: `node-${i}-${j - 1}`,
          target: `node-${i}-${j}`,
          label: ['spawned', 'accessed', 'connected', 'modified'][
            Math.floor(Math.random() * 4)
          ],
        });
      }
    }

    chains.push({
      id: `chain-${String(i + 1).padStart(3, '0')}`,
      severity: severities[Math.floor(Math.random() * severities.length)],
      created_at: dayjs()
        .subtract(i * 2, 'hour')
        .toISOString(),
      nodes,
      edges,
    });
  }

  return chains;
};
