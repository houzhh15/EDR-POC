/**
 * @file module_manager.h
 * @brief 模块化插件架构 - 模块管理器
 *
 * 支持采集器、检测器、响应器等模块的动态注册和生命周期管理。
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#ifndef EDR_MODULE_MANAGER_H
#define EDR_MODULE_MANAGER_H

#ifdef __cplusplus
extern "C" {
#endif

#include "edr_core.h"
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

/* ============================================================
 * 模块类型定义
 * ============================================================ */

/**
 * @brief 模块类型枚举
 */
typedef enum edr_module_type {
    EDR_MODULE_COLLECTOR = 0,   /**< 采集器模块 */
    EDR_MODULE_DETECTOR = 1,    /**< 检测器模块 */
    EDR_MODULE_RESPONDER = 2,   /**< 响应器模块 */
} edr_module_type_t;

/* ============================================================
 * 模块操作接口
 * ============================================================ */

/**
 * @brief 模块操作接口结构
 */
typedef struct edr_module_ops {
    const char* name;           /**< 模块名称 (唯一标识) */
    const char* version;        /**< 模块版本 */
    edr_module_type_t type;     /**< 模块类型 */
    
    /**
     * @brief 模块初始化
     * @param config 配置参数 (可为 NULL)
     * @return EDR_OK 成功，其他值表示失败
     */
    edr_error_t (*init)(void* config);
    
    /**
     * @brief 启动模块
     * @return EDR_OK 成功，其他值表示失败
     */
    edr_error_t (*start)(void);
    
    /**
     * @brief 停止模块
     * @return EDR_OK 成功，其他值表示失败
     */
    edr_error_t (*stop)(void);
    
    /**
     * @brief 清理模块资源
     */
    void (*cleanup)(void);
} edr_module_ops_t;

/* ============================================================
 * 模块管理器接口
 * ============================================================ */

/**
 * @brief 初始化模块管理器
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_module_manager_init(void);

/**
 * @brief 清理模块管理器
 */
void edr_module_manager_cleanup(void);

/**
 * @brief 注册模块
 * @param ops 模块操作接口
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_module_register(const edr_module_ops_t* ops);

/**
 * @brief 注销模块
 * @param name 模块名称
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_module_unregister(const char* name);

/**
 * @brief 获取模块
 * @param name 模块名称
 * @return 模块操作接口，未找到返回 NULL
 */
const edr_module_ops_t* edr_module_get(const char* name);

/**
 * @brief 列出指定类型的模块
 * @param type 模块类型
 * @param list 输出数组
 * @param max_count 数组最大容量
 * @param actual_count 实际模块数 (可为 NULL)
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_module_list(edr_module_type_t type, const edr_module_ops_t** list, 
                            size_t max_count, size_t* actual_count);

/**
 * @brief 启动所有已注册模块
 * @param config 配置参数 (传递给每个模块的 init)
 * @return EDR_OK 全部成功，其他值表示有模块启动失败
 */
edr_error_t edr_module_start_all(void* config);

/**
 * @brief 停止所有已注册模块
 * @return EDR_OK 全部成功，其他值表示有模块停止失败
 */
edr_error_t edr_module_stop_all(void);

/**
 * @brief 获取已注册模块数量
 * @return 模块数量
 */
size_t edr_module_count(void);

#ifdef __cplusplus
}
#endif

#endif /* EDR_MODULE_MANAGER_H */
