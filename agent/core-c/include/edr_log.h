/**
 * @file edr_log.h
 * @brief EDR日志框架
 * 
 * 提供统一的日志接口,支持多级别日志、文件输出、级别过滤等功能。
 * 
 * 核心功能:
 * - 4个日志级别: DEBUG, INFO, WARN, ERROR
 * - 统一输出格式: 时间戳-级别-文件:行号-消息
 * - 级别过滤(配置最小级别)
 * - 支持输出到stdout或文件
 * - 便捷的宏定义封装
 * 
 * 使用示例:
 * ```c
 * EDR_LOG_INFO("ETW Session started: handle=%p", session_handle);
 * EDR_LOG_ERROR("Failed to open process: pid=%d, error=%d", pid, GetLastError());
 * ```
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifndef EDR_LOG_H
#define EDR_LOG_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdio.h>

/**
 * @brief 日志级别枚举
 */
typedef enum {
    EDR_LOG_LEVEL_DEBUG = 0,  // 详细调试信息
    EDR_LOG_LEVEL_INFO  = 1,  // 关键操作信息
    EDR_LOG_LEVEL_WARN  = 2,  // 可恢复的警告
    EDR_LOG_LEVEL_ERROR = 3   // 严重错误
} edr_log_level_t;

/**
 * @brief 日志输出目标
 */
typedef enum {
    EDR_LOG_TARGET_STDOUT = 0,  // 标准输出
    EDR_LOG_TARGET_FILE   = 1   // 文件输出
} edr_log_target_t;

/**
 * @brief 日志配置结构体
 */
typedef struct {
    edr_log_level_t min_level;     // 最小日志级别(低于此级别的日志不输出)
    edr_log_target_t target;       // 输出目标
    char log_file_path[260];       // 日志文件路径
    FILE* log_file_handle;         // 日志文件句柄
} edr_log_config_t;

/**
 * @brief 初始化日志系统
 * 
 * @param min_level 最小日志级别
 * @param target 输出目标(stdout或文件)
 * @param log_file_path 日志文件路径(如果target=FILE)
 * @return 0=成功, 非0=失败
 */
int edr_log_init(edr_log_level_t min_level, edr_log_target_t target, const char* log_file_path);

/**
 * @brief 关闭日志系统
 */
void edr_log_shutdown(void);

/**
 * @brief 核心日志函数
 * 
 * 输出格式: [时间戳] [级别] [文件:行号] 消息
 * 
 * @param level 日志级别
 * @param file 源文件名(__FILE__)
 * @param line 行号(__LINE__)
 * @param format printf风格的格式字符串
 * @param ... 可变参数
 */
void edr_log(edr_log_level_t level, const char* file, int line, const char* format, ...);

/**
 * @brief 便捷日志宏定义
 * 
 * 自动填充__FILE__和__LINE__参数。
 */
#define EDR_LOG_DEBUG(fmt, ...) \
    edr_log(EDR_LOG_LEVEL_DEBUG, __FILE__, __LINE__, fmt, ##__VA_ARGS__)

#define EDR_LOG_INFO(fmt, ...) \
    edr_log(EDR_LOG_LEVEL_INFO, __FILE__, __LINE__, fmt, ##__VA_ARGS__)

#define EDR_LOG_WARN(fmt, ...) \
    edr_log(EDR_LOG_LEVEL_WARN, __FILE__, __LINE__, fmt, ##__VA_ARGS__)

#define EDR_LOG_ERROR(fmt, ...) \
    edr_log(EDR_LOG_LEVEL_ERROR, __FILE__, __LINE__, fmt, ##__VA_ARGS__)

#ifdef __cplusplus
}
#endif

#endif /* EDR_LOG_H */
