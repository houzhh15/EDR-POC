import React from 'react';
import { Card, Empty, Spin, Tooltip } from 'antd';
import { useNavigate } from 'react-router-dom';
import { HeatmapChart } from '../../components/charts';
import { useMitreCoverage } from '../../hooks/useDashboard';

/**
 * MITRE ATT&CK热力图组件
 * 展示ATT&CK框架的战术和技术覆盖度
 */
export const MitreHeatmap: React.FC = () => {
  const navigate = useNavigate();
  const { data, isLoading, error } = useMitreCoverage();

  if (isLoading) {
    return (
      <Card title="MITRE ATT&CK 覆盖度热力图" style={{ height: '100%' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            height: 400,
          }}
        >
          <Spin />
        </div>
      </Card>
    );
  }

  if (error || !data || data.length === 0) {
    return (
      <Card title="MITRE ATT&CK 覆盖度热力图" style={{ height: '100%' }}>
        <Empty description="加载失败或无数据" />
      </Card>
    );
  }

  // 提取唯一的战术列表（X轴）
  const tactics = Array.from(new Set(data.map((cell) => cell.tactic)));

  // 提取唯一的技术ID列表（Y轴）
  const techniques = Array.from(new Set(data.map((cell) => cell.technique_id)));

  // 创建技术ID到名称的映射
  const techniqueNameMap = new Map(
    data.map((cell) => [cell.technique_id, cell.technique_name])
  );

  // 转换数据为热力图格式: [tacticIndex, techniqueIndex, hit_count]
  const heatmapData: [number, number, number][] = data.map((cell) => {
    const tacticIndex = tactics.indexOf(cell.tactic);
    const techniqueIndex = techniques.indexOf(cell.technique_id);
    return [tacticIndex, techniqueIndex, cell.hit_count];
  });

  return (
    <Card
      title="MITRE ATT&CK 覆盖度热力图"
      style={{ height: '100%' }}
      extra={
        <Tooltip title="点击单元格查看相关规则">
          <span style={{ fontSize: '12px', color: '#8c8c8c', cursor: 'help' }}>
            覆盖 {techniques.length} 个技术，{tactics.length} 个战术
          </span>
        </Tooltip>
      }
    >
      <div style={{ paddingBottom: '40px' }}>
        <HeatmapChart
          data={heatmapData}
          xCategories={tactics}
          yCategories={techniques.map(
            (id) => `${id}\n${techniqueNameMap.get(id)?.substring(0, 20) || ''}`
          )}
          colorRange={['#f0f9ff', '#0ea5e9', '#0369a1']}
          height={500}
          onClick={(params) => {
            // 点击单元格处理
            const techniqueId = techniques[parseInt(params.y)];
            navigate(`/rules?technique=${techniqueId}`);
          }}
        />
      </div>
    </Card>
  );
};
