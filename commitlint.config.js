// ============================================================
// EDR Platform - Commit Message 规范配置
// ============================================================
// 使用 Conventional Commits 规范
// https://www.conventionalcommits.org/
// ============================================================

module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // 类型枚举
    'type-enum': [
      2,
      'always',
      [
        'feat',     // 新功能
        'fix',      // 修复 Bug
        'docs',     // 文档更新
        'style',    // 代码格式化（不影响功能）
        'refactor', // 代码重构
        'perf',     // 性能优化
        'test',     // 测试相关
        'build',    // 构建系统或外部依赖
        'ci',       // CI 配置
        'chore',    // 其他变更
        'revert',   // 回滚
      ],
    ],
    // 主题最大长度
    'subject-max-length': [2, 'always', 72],
    // 主题不能为空
    'subject-empty': [2, 'never'],
    // 类型不能为空
    'type-empty': [2, 'never'],
  },
};
