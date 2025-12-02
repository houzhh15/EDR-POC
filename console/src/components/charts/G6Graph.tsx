/**
 * G6Graph - 使用官方Mermaid库渲染流程图
 * 提供美观的关系图可视化
 */
import React, { useEffect, useRef, useMemo } from 'react';
import mermaid from 'mermaid';
import { Empty, Spin } from 'antd';
import type { AttackChain } from '../../types/dashboard';

// 初始化Mermaid配置
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  securityLevel: 'loose',
  flowchart: {
    useMaxWidth: true,
    htmlLabels: true,
    curve: 'basis',
    nodeSpacing: 30,
    rankSpacing: 40,
    padding: 10,
  },
  fontSize: 11,
});

/**
 * G6Graph节点
 */
export interface G6Node {
  id: string;
  type?: string;
  label?: string;
  [key: string]: any;
}

/**
 * G6Graph边
 */
export interface G6Edge {
  source: string;
  target: string;
  label?: string;
  [key: string]: any;
}

/**
 * G6Graph Props
 */
export interface G6GraphProps {
  /** 节点数组 */
  nodes: G6Node[];
  /** 边数组 */
  edges: G6Edge[];
  /** 高度 */
  height?: string | number;
  /** 布局类型 */
  layout?: 'force' | 'dagre';
  /** 节点点击回调 */
  onNodeClick?: (nodeId: string) => void;
  /** 选中节点ID */
  selectedNodeId?: string | null;
  /** 加载状态 */
  loading?: boolean;
}

/**
 * 生成Mermaid流程图语法
 */
function generateMermaidChart(nodes: G6Node[], edges: G6Edge[]): string {
  // 节点类型样式映射
  const nodeStyleMap: Record<string, string> = {
    process: '([{label}])',      // 体育场形 - 进程
    file: '[{label}]',            // 矩形 - 文件
    network: '{{{label}}}',       // 六边形 - 网络
    host: '({label})',            // 圆形 - 主机
    user: '[/{label}/]',          // 梯形 - 用户
  };

  let chart = 'graph TB\n';
  
  // 定义节点样式类（缩小尺寸）
  chart += '  classDef processStyle fill:#1890ff,stroke:#096dd9,stroke-width:1px,color:#fff,font-size:11px\n';
  chart += '  classDef fileStyle fill:#52c41a,stroke:#389e0d,stroke-width:1px,color:#fff,font-size:11px\n';
  chart += '  classDef networkStyle fill:#fa8c16,stroke:#d46b08,stroke-width:1px,color:#fff,font-size:11px\n';
  chart += '  classDef hostStyle fill:#eb2f96,stroke:#c41d7f,stroke-width:1px,color:#fff,font-size:11px\n';
  chart += '  classDef userStyle fill:#722ed1,stroke:#531dab,stroke-width:1px,color:#fff,font-size:11px\n';
  
  // 添加节点定义
  nodes.forEach(node => {
    const nodeId = node.id.replace(/[^a-zA-Z0-9]/g, '_'); // 清理ID
    const label = (node.label || node.id).substring(0, 12); // 限制标签长度（缩短）
    const nodeType = node.type || 'process';
    const template = nodeStyleMap[nodeType] || '[{label}]';
    const nodeDefinition = template.replace('{label}', label);
    
    chart += `  ${nodeId}${nodeDefinition}\n`;
    
    // 应用样式类
    const styleClass = `${nodeType}Style`;
    chart += `  class ${nodeId} ${styleClass}\n`;
  });
  
  // 添加边定义
  edges.forEach(edge => {
    const sourceId = edge.source.replace(/[^a-zA-Z0-9]/g, '_');
    const targetId = edge.target.replace(/[^a-zA-Z0-9]/g, '_');
    const label = edge.label ? `|${edge.label}|` : '';
    chart += `  ${sourceId} -->${label} ${targetId}\n`;
  });
  
  return chart;
}

/**
 * G6Graph组件
 * @description 使用Mermaid流程图展示关系图
 */
export const G6Graph: React.FC<G6GraphProps> = ({
  nodes,
  edges,
  height = 500,
  loading = false,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const mermaidChart = useMemo(() => {
    if (nodes.length === 0) return '';
    return generateMermaidChart(nodes, edges);
  }, [nodes, edges]);

  useEffect(() => {
    if (containerRef.current && mermaidChart) {
      // 清空容器
      containerRef.current.innerHTML = '';
      
      // 创建一个div用于渲染Mermaid图表
      const graphDiv = document.createElement('div');
      graphDiv.className = 'mermaid';
      graphDiv.textContent = mermaidChart;
      containerRef.current.appendChild(graphDiv);
      
      // 渲染Mermaid图表
      mermaid.run({
        nodes: [graphDiv],
      }).catch((error) => {
        console.error('Mermaid rendering error:', error);
      });
    }
  }, [mermaidChart]);

  if (loading) {
    return (
      <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description="暂无数据" />
      </div>
    );
  }

  return (
    <div 
      ref={containerRef} 
      style={{ 
        width: '100%', 
        minHeight: height,
        overflow: 'auto', 
        padding: '20px',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'flex-start',
      }}
    />
  );
};

/**
 * AttackChainGraph Props
 */
export interface AttackChainGraphProps {
  /** 攻击链数据 */
  chain: AttackChain | null;
  /** 节点点击回调 */
  onNodeClick?: (nodeId: string) => void;
  /** 高度 */
  height?: string | number;
}

/**
 * AttackChainGraph组件
 * @description 攻击链可视化（使用Mermaid流程图）
 */
export const AttackChainGraph: React.FC<AttackChainGraphProps> = ({
  chain,
  height = 700,
}) => {
  if (!chain) {
    return (
      <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description="暂无攻击链数据" />
      </div>
    );
  }

  return (
    <G6Graph
      nodes={chain.nodes}
      edges={chain.edges}
      height={height}
    />
  );
};
