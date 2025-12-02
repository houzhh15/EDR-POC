/**
 * @file edr_events.h
 * @brief EDR进程事件数据结构定义
 * @version 1.0
 * @date 2025-12-02
 * 
 * 定义Windows进程事件采集的核心数据结构，用于ETW事件的标准化表示
 */

#ifndef EDR_EVENTS_H
#define EDR_EVENTS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>  // for size_t

// Windows MAX_PATH定义
#ifndef MAX_PATH
#define MAX_PATH 260
#endif

/**
 * @brief 进程事件类型枚举
 */
typedef enum {
    EDR_PROCESS_START = 1,  /**< 进程创建事件 */
    EDR_PROCESS_END = 2     /**< 进程终止事件 */
} edr_process_event_type_t;

/**
 * @brief 进程事件结构体
 * 
 * 包含进程完整上下文信息，用于安全检测和审计
 * 结构体大小约6KB，优化内存对齐
 */
typedef struct {
    uint64_t timestamp;                   /**< 事件时间戳(UTC,100ns单位) */
    uint32_t pid;                         /**< 进程ID */
    uint32_t ppid;                        /**< 父进程ID */
    char process_name[256];               /**< 进程名称 */
    char executable_path[MAX_PATH];       /**< 可执行文件完整路径 */
    char command_line[4096];              /**< 命令行参数 */
    char username[128];                   /**< 用户名 */
    uint8_t sha256[32];                   /**< 文件SHA256哈希 */
    edr_process_event_type_t event_type;  /**< 事件类型 */
    int32_t exit_code;                    /**< 退出码(仅PROCESS_END事件) */
    uint32_t reserved[4];                 /**< 预留字段用于未来扩展 */
} edr_process_event_t;

/**
 * @brief 获取事件结构体大小（用于验证）
 * @return 结构体大小（字节）
 */
static inline size_t edr_get_event_struct_size(void) {
    return sizeof(edr_process_event_t);
}

#ifdef __cplusplus
}
#endif

#endif /* EDR_EVENTS_H */
