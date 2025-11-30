/**
 * @file edr_core.h
 * @brief EDR Core Library - 公共头文件
 *
 * 此头文件定义了 EDR 核心库的公共接口，供 Go 主程序通过 CGO 调用。
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#ifndef EDR_CORE_H
#define EDR_CORE_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

/* ============================================================
 * 版本信息
 * ============================================================ */

#define EDR_CORE_VERSION_MAJOR 0
#define EDR_CORE_VERSION_MINOR 1
#define EDR_CORE_VERSION_PATCH 0

/**
 * @brief 获取库版本字符串
 * @return 版本字符串 (如 "0.1.0")
 */
const char* edr_core_version(void);

/* ============================================================
 * 错误码定义
 * ============================================================ */

typedef enum {
    EDR_OK = 0,                    /**< 成功 */
    EDR_ERR_UNKNOWN = -1,          /**< 未知错误 */
    EDR_ERR_INVALID_PARAM = -2,    /**< 无效参数 */
    EDR_ERR_NO_MEMORY = -3,        /**< 内存不足 */
    EDR_ERR_NOT_INITIALIZED = -4,  /**< 未初始化 */
    EDR_ERR_ALREADY_INITIALIZED = -5, /**< 已初始化 */
    EDR_ERR_PERMISSION = -6,       /**< 权限不足 */
    EDR_ERR_NOT_SUPPORTED = -7,    /**< 不支持 */
    EDR_ERR_TIMEOUT = -8,          /**< 超时 */
} edr_error_t;

/**
 * @brief 获取错误描述
 * @param err 错误码
 * @return 错误描述字符串
 */
const char* edr_error_string(edr_error_t err);

/* ============================================================
 * 核心初始化/销毁
 * ============================================================ */

/**
 * @brief 初始化 EDR 核心库
 *
 * 必须在调用其他函数前调用此函数。
 *
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_core_init(void);

/**
 * @brief 销毁 EDR 核心库
 *
 * 释放所有资源，停止所有采集任务。
 */
void edr_core_cleanup(void);

/**
 * @brief 检查库是否已初始化
 * @return true 已初始化，false 未初始化
 */
bool edr_core_is_initialized(void);

/* ============================================================
 * 采集器接口 (Collector)
 * ============================================================ */

/**
 * @brief 事件回调函数类型
 *
 * @param event_type 事件类型
 * @param data 事件数据 (JSON 格式)
 * @param data_len 数据长度
 * @param user_data 用户数据
 */
typedef void (*edr_event_callback_t)(
    uint32_t event_type,
    const char* data,
    size_t data_len,
    void* user_data
);

/**
 * @brief 启动事件采集
 *
 * @param callback 事件回调函数
 * @param user_data 传递给回调的用户数据
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_collector_start(edr_event_callback_t callback, void* user_data);

/**
 * @brief 停止事件采集
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_collector_stop(void);

/**
 * @brief 检查采集器是否正在运行
 * @return true 运行中，false 未运行
 */
bool edr_collector_is_running(void);

/* ============================================================
 * 检测器接口 (Detector)
 * ============================================================ */

/**
 * @brief 加载 YARA 规则文件
 *
 * @param rules_path 规则文件路径
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_detector_load_yara_rules(const char* rules_path);

/**
 * @brief 使用 YARA 规则扫描数据
 *
 * @param data 待扫描数据
 * @param data_len 数据长度
 * @param matches 匹配结果 (JSON 格式，调用者负责释放)
 * @return 匹配的规则数量，负值表示错误
 */
int edr_detector_scan_yara(const void* data, size_t data_len, char** matches);

/**
 * @brief 释放检测结果内存
 * @param matches 由 edr_detector_scan_* 返回的结果
 */
void edr_detector_free_matches(char* matches);

/* ============================================================
 * 响应执行接口 (Response)
 * ============================================================ */

/**
 * @brief 终止指定进程
 *
 * @param pid 进程 ID
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_response_kill_process(uint32_t pid);

/**
 * @brief 隔离文件
 *
 * @param file_path 文件路径
 * @param quarantine_path 隔离目录
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t edr_response_quarantine_file(const char* file_path, const char* quarantine_path);

#ifdef __cplusplus
}
#endif

#endif /* EDR_CORE_H */
