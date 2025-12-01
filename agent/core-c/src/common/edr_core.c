/**
 * @file edr_core.c
 * @brief EDR Core Library - 核心实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "edr_core.h"
#include "pal.h"
#include "ring_buffer.h"
#include "module_manager.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* ============================================================
 * 内部状态
 * ============================================================ */

static bool g_initialized = false;
static bool g_collector_running = false;

/* 全局事件队列 */
static edr_ring_buffer_t* g_event_queue = NULL;
#define EDR_DEFAULT_QUEUE_CAPACITY 16384  /* 必须是 2 的幂 */

/* ============================================================
 * 版本信息
 * ============================================================ */

const char* edr_core_version(void) {
    static char version[32];
    snprintf(version, sizeof(version), "%d.%d.%d",
             EDR_CORE_VERSION_MAJOR,
             EDR_CORE_VERSION_MINOR,
             EDR_CORE_VERSION_PATCH);
    return version;
}

/* ============================================================
 * 错误处理
 * ============================================================ */

const char* edr_error_string(edr_error_t err) {
    switch (err) {
        case EDR_OK:
            return "Success";
        case EDR_ERR_UNKNOWN:
            return "Unknown error";
        case EDR_ERR_INVALID_PARAM:
            return "Invalid parameter";
        case EDR_ERR_NO_MEMORY:
            return "Out of memory";
        case EDR_ERR_NOT_INITIALIZED:
            return "Not initialized";
        case EDR_ERR_ALREADY_INITIALIZED:
            return "Already initialized";
        case EDR_ERR_PERMISSION:
            return "Permission denied";
        case EDR_ERR_NOT_SUPPORTED:
            return "Not supported";
        case EDR_ERR_TIMEOUT:
            return "Timeout";
        default:
            return "Unknown error code";
    }
}

/* ============================================================
 * 核心初始化/销毁
 * ============================================================ */

edr_error_t edr_core_init(void) {
    if (g_initialized) {
        return EDR_ERR_ALREADY_INITIALIZED;
    }

    edr_error_t err;

    /* 1. 初始化平台抽象层 */
    err = pal_init();
    if (err != EDR_OK) {
        return err;
    }

    /* 2. 初始化模块管理器 */
    err = edr_module_manager_init();
    if (err != EDR_OK) {
        pal_cleanup();
        return err;
    }

    /* 3. 创建全局事件队列 */
    g_event_queue = ring_buffer_create(EDR_DEFAULT_QUEUE_CAPACITY);
    if (g_event_queue == NULL) {
        edr_module_manager_cleanup();
        pal_cleanup();
        return EDR_ERR_NO_MEMORY;
    }

    g_initialized = true;
    return EDR_OK;
}

void edr_core_cleanup(void) {
    if (!g_initialized) {
        return;
    }

    /* 停止采集器 */
    if (g_collector_running) {
        edr_collector_stop();
    }

    /* 按逆序清理 */

    /* 1. 停止所有模块 */
    edr_module_stop_all();

    /* 2. 销毁事件队列 */
    if (g_event_queue != NULL) {
        ring_buffer_destroy(g_event_queue);
        g_event_queue = NULL;
    }

    /* 3. 清理模块管理器 */
    edr_module_manager_cleanup();

    /* 4. 清理平台层 */
    pal_cleanup();

    g_initialized = false;
}

bool edr_core_is_initialized(void) {
    return g_initialized;
}

/* ============================================================
 * 采集器接口 (占位实现)
 * ============================================================ */

edr_error_t edr_collector_start(edr_event_callback_t callback, void* user_data) {
    (void)callback;   /* 避免未使用警告 */
    (void)user_data;

    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }

    if (g_collector_running) {
        return EDR_ERR_ALREADY_INITIALIZED;
    }

    /* TODO: 实现平台相关的采集逻辑
     * - Windows: ETW
     * - Linux: eBPF
     * - macOS: Endpoint Security
     */

    g_collector_running = true;
    return EDR_OK;
}

edr_error_t edr_collector_stop(void) {
    if (!g_collector_running) {
        return EDR_OK;
    }

    /* TODO: 停止平台相关的采集 */

    g_collector_running = false;
    return EDR_OK;
}

bool edr_collector_is_running(void) {
    return g_collector_running;
}

/* ============================================================
 * 检测器接口 (占位实现)
 * ============================================================ */

edr_error_t edr_detector_load_yara_rules(const char* rules_path) {
    if (rules_path == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }

    /* TODO: 使用 libyara 加载规则 */

    return EDR_OK;
}

int edr_detector_scan_yara(const void* data, size_t data_len, char** matches) {
    if (data == NULL || matches == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }

    (void)data_len;

    /* TODO: 使用 libyara 扫描 */

    *matches = NULL;
    return 0;  /* 无匹配 */
}

void edr_detector_free_matches(char* matches) {
    if (matches != NULL) {
        free(matches);
    }
}

/* ============================================================
 * 响应执行接口 (占位实现)
 * ============================================================ */

edr_error_t edr_response_kill_process(uint32_t pid) {
    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }

    (void)pid;

    /* TODO: 实现平台相关的进程终止 */

    return EDR_ERR_NOT_SUPPORTED;
}

edr_error_t edr_response_quarantine_file(const char* file_path, const char* quarantine_path) {
    if (file_path == NULL || quarantine_path == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }

    /* TODO: 实现文件隔离 */

    return EDR_ERR_NOT_SUPPORTED;
}

/* ============================================================
 * 事件队列访问接口
 * ============================================================ */

void* edr_core_get_event_queue(void) {
    return (void*)g_event_queue;
}
