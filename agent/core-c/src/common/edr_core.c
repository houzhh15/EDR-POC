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
#include "edr_errors.h"
#include "edr_events.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include "../collector/windows/etw_session.h"
#include "../collector/windows/etw_process.h"
#endif

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

/* ============================================================
 * CGO桥接API - Windows进程事件采集器
 * ============================================================ */

#ifdef _WIN32
#include "../collector/windows/etw_session.h"
#include "../collector/windows/etw_process.h"
#include "event_buffer.h"

/**
 * @brief Session句柄结构体(内部使用)
 */
typedef struct {
    etw_session_t* session;
    etw_process_consumer_t* consumer;
} edr_collector_session_t;

/**
 * @brief 启动进程事件采集器(CGO接口)
 * 
 * @param out_handle 输出Session句柄指针
 * @return 0=成功, 非0=错误码
 */
int edr_start_process_collector(void** out_handle) {
    if (!g_initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    if (out_handle == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    // 获取全局事件队列
    event_buffer_t* queue = (event_buffer_t*)edr_core_get_event_queue();
    if (queue == NULL) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    // 分配Session结构
    edr_collector_session_t* session = (edr_collector_session_t*)calloc(1, sizeof(edr_collector_session_t));
    if (session == NULL) {
        return EDR_ERR_NO_MEMORY;
    }
    
    // 创建进程事件消费者
    session->consumer = etw_process_consumer_create(queue);
    if (session->consumer == NULL) {
        free(session);
        return EDR_ERR_NO_MEMORY;
    }
    
    // 初始化ETW Session
    session->session = etw_session_init(ETW_SESSION_NAME);
    if (session->session == NULL) {
        etw_process_consumer_destroy(session->consumer);
        free(session);
        return EDR_ERR_NO_MEMORY;
    }
    
    // 启动ETW Session(传递consumer的回调)
    int result = etw_session_start(
        session->session,
        (event_callback_fn)etw_process_event_callback,
        session->consumer
    );
    
    if (result != EDR_SUCCESS) {
        etw_session_destroy(session->session);
        etw_process_consumer_destroy(session->consumer);
        free(session);
        return result;
    }
    
    *out_handle = session;
    
    return EDR_SUCCESS;
}

/**
 * @brief 停止进程事件采集器(CGO接口)
 * 
 * @param handle Session句柄
 * @return 0=成功, 非0=错误码
 */
int edr_stop_process_collector(void* handle) {
    if (handle == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    edr_collector_session_t* session = (edr_collector_session_t*)handle;
    
    // 停止ETW Session
    if (session->session != NULL) {
        etw_session_stop(session->session);
        etw_session_destroy(session->session);
    }
    
    // 销毁Consumer
    if (session->consumer != NULL) {
        etw_process_consumer_destroy(session->consumer);
    }
    
    free(session);
    
    return EDR_SUCCESS;
}

/**
 * @brief 轮询进程事件(CGO接口,批量获取)
 * 
 * @param handle Session句柄
 * @param events 事件数组指针(调用者分配)
 * @param max_count 最大获取数量
 * @param out_count 实际获取数量
 * @return 0=成功, 非0=错误码
 */
int edr_poll_process_events(
    void* handle,
    edr_process_event_t* events,
    int max_count,
    int* out_count
) {
    if (handle == NULL || events == NULL || out_count == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (max_count <= 0) {
        *out_count = 0;
        return EDR_SUCCESS;
    }
    
    // 从全局队列批量pop事件
    event_buffer_t* queue = (event_buffer_t*)edr_core_get_event_queue();
    if (queue == NULL) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    int count = event_buffer_pop_batch(queue, events, max_count);
    *out_count = count;
    
    return EDR_SUCCESS;
}

#else  /* !_WIN32 */

/**
 * @brief 启动进程事件采集器(CGO接口) - 非Windows平台
 */
int edr_start_process_collector(void** out_handle) {
    (void)out_handle;
    return EDR_ERROR_NOT_SUPPORTED;
}

/**
 * @brief 停止进程事件采集器(CGO接口) - 非Windows平台
 */
int edr_stop_process_collector(void* handle) {
    (void)handle;
    return EDR_ERROR_NOT_SUPPORTED;
}

/**
 * @brief 轮询进程事件(CGO接口) - 非Windows平台
 */
int edr_poll_process_events(
    void* handle,
    edr_process_event_t* events,
    int max_count,
    int* out_count
) {
    (void)handle;
    (void)events;
    (void)max_count;
    (void)out_count;
    return EDR_ERROR_NOT_SUPPORTED;
}

#endif  /* _WIN32 */
