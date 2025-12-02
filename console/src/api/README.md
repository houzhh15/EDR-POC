# Dashboard API 使用说明

## 🔧 开发模式（当前）

当前 `dashboard.ts` 使用 **Mock数据模式**，因为后端API尚未实现。

### Mock数据特性
- ✅ 模拟300-500ms网络延迟
- ✅ 随机生成真实感数据（告警、终端、威胁、MITRE、攻击链）
- ✅ 时间序列数据（支持24h/7d/30d）
- ✅ 完整TypeScript类型支持

### 文件结构
```
api/
├── dashboard.ts          # API封装（当前使用Mock）
├── __mocks__/
│   └── dashboard.ts      # Mock数据生成器
└── README.md             # 本文档
```

---

## 🚀 生产模式切换

当后端 Dashboard API 实现后，按以下步骤切换：

### 步骤1: 修改 `dashboard.ts` 导入
```typescript
// 注释掉Mock导入
// import * as mockApi from './__mocks__/dashboard';

// 启用真实API导入
import { apiClient } from './client';
```

### 步骤2: 修改函数实现
将所有函数从：
```typescript
async function getDashboardStats(): Promise<DashboardStats> {
  return mockApi.getDashboardStats();
}
```

改为：
```typescript
async function getDashboardStats(): Promise<DashboardStats> {
  try {
    const response = await apiClient.get<DashboardStats>('/v1/dashboard/stats');
    return response.data;
  } catch (error) {
    console.error('获取仪表盘统计数据失败:', error);
    throw error;
  }
}
```

### 步骤3: 后端API端点
确保后端实现以下5个接口：

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 统计数据 | GET | `/api/v1/dashboard/stats` | 告警/终端/规则/事件统计 |
| 告警趋势 | GET | `/api/v1/dashboard/alert-trend?range=24h` | 时间序列趋势 |
| Top威胁 | GET | `/api/v1/dashboard/top-threats?type=process&limit=10` | Top N分析 |
| MITRE覆盖 | GET | `/api/v1/dashboard/mitre-coverage` | ATT&CK矩阵 |
| 攻击链 | GET | `/api/v1/dashboard/attack-chains?limit=5` | 攻击链数据 |

---

## 📊 Mock数据示例

### 统计数据
```json
{
  "alerts": { "critical": 5, "high": 12, "medium": 45, "low": 128, "total": 190 },
  "endpoints": { "online": 856, "offline": 23, "at_risk": 15, "total": 894 },
  "rules": { "enabled": 156, "disabled": 12, "total": 168 },
  "events_today": 12456,
  "mttd_minutes": 8.5,
  "mttr_minutes": 45.2
}
```

### Top威胁（process类型）
```json
[
  { "rank": 1, "name": "powershell.exe", "count": 156, "risk_level": "high" },
  { "rank": 2, "name": "cmd.exe", "count": 128, "risk_level": "medium" },
  ...
]
```

---

## 🧪 测试

### 单元测试（使用Mock数据）
```bash
npm run test
```

Vitest会自动识别 `__mocks__` 目录，使用Mock数据进行测试。

### 集成测试（需要真实后端）
```bash
# 切换到生产模式后
npm run test:integration
```

---

## 📝 注意事项

1. **不要删除Mock文件**: 即使切换到生产模式，保留 `__mocks__/dashboard.ts` 用于测试
2. **数据格式一致**: 确保后端API返回的数据格式与Mock数据一致
3. **错误处理**: 真实API需要统一错误处理（401跳转登录、404提示、500重试）
4. **类型定义**: 如果后端响应格式变更，同步更新 `types/dashboard.ts`

---

## 🔄 回滚到Mock模式

如果生产API出现问题，可快速回滚：

```typescript
// 恢复Mock导入
import * as mockApi from './__mocks__/dashboard';

// 注释真实API导入
// import { apiClient } from './client';

// 恢复简单实现
async function getDashboardStats(): Promise<DashboardStats> {
  return mockApi.getDashboardStats();
}
```

---

**最后更新**: 2025-12-02  
**当前模式**: Mock数据模式  
**待办**: 后端Dashboard API实现（预计需要2-3天）
