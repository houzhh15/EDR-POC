/**
 * @file module_manager.c
 * @brief 模块化插件架构 - 模块管理器实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "module_manager.h"
#include <stdlib.h>
#include <string.h>

/* ============================================================
 * 内部常量和状态
 * ============================================================ */

#define MAX_MODULES 32

/** 模块条目 */
typedef struct module_entry {
    const edr_module_ops_t* ops;    /**< 模块操作接口 */
    bool initialized;                /**< 是否已初始化 */
    bool running;                    /**< 是否正在运行 */
} module_entry_t;

/** 模块管理器状态 */
static struct {
    module_entry_t modules[MAX_MODULES];
    size_t count;
    bool initialized;
} g_manager = {0};

/* ============================================================
 * 内部辅助函数
 * ============================================================ */

/**
 * @brief 查找模块索引
 * @return 模块索引，未找到返回 -1
 */
static int find_module_index(const char* name) {
    if (name == NULL) {
        return -1;
    }
    
    for (size_t i = 0; i < g_manager.count; i++) {
        if (g_manager.modules[i].ops != NULL && 
            strcmp(g_manager.modules[i].ops->name, name) == 0) {
            return (int)i;
        }
    }
    
    return -1;
}

/* ============================================================
 * 模块管理器接口实现
 * ============================================================ */

edr_error_t edr_module_manager_init(void) {
    if (g_manager.initialized) {
        return EDR_ERR_ALREADY_INITIALIZED;
    }
    
    memset(&g_manager, 0, sizeof(g_manager));
    g_manager.initialized = true;
    
    return EDR_OK;
}

void edr_module_manager_cleanup(void) {
    if (!g_manager.initialized) {
        return;
    }
    
    /* 停止所有运行中的模块 */
    edr_module_stop_all();
    
    /* 清理所有模块 */
    for (size_t i = 0; i < g_manager.count; i++) {
        if (g_manager.modules[i].ops != NULL) {
            if (g_manager.modules[i].initialized && g_manager.modules[i].ops->cleanup != NULL) {
                g_manager.modules[i].ops->cleanup();
            }
            g_manager.modules[i].ops = NULL;
            g_manager.modules[i].initialized = false;
            g_manager.modules[i].running = false;
        }
    }
    
    g_manager.count = 0;
    g_manager.initialized = false;
}

edr_error_t edr_module_register(const edr_module_ops_t* ops) {
    if (!g_manager.initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    if (ops == NULL || ops->name == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    /* 检查是否已存在同名模块 */
    if (find_module_index(ops->name) >= 0) {
        return EDR_ERR_ALREADY_INITIALIZED;
    }
    
    /* 检查容量 */
    if (g_manager.count >= MAX_MODULES) {
        return EDR_ERR_NO_MEMORY;
    }
    
    /* 添加模块 */
    g_manager.modules[g_manager.count].ops = ops;
    g_manager.modules[g_manager.count].initialized = false;
    g_manager.modules[g_manager.count].running = false;
    g_manager.count++;
    
    return EDR_OK;
}

edr_error_t edr_module_unregister(const char* name) {
    if (!g_manager.initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    int index = find_module_index(name);
    if (index < 0) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    module_entry_t* entry = &g_manager.modules[index];
    
    /* 停止模块 */
    if (entry->running && entry->ops->stop != NULL) {
        entry->ops->stop();
        entry->running = false;
    }
    
    /* 清理模块 */
    if (entry->initialized && entry->ops->cleanup != NULL) {
        entry->ops->cleanup();
        entry->initialized = false;
    }
    
    /* 移除模块 (移动后面的元素) */
    for (size_t i = (size_t)index; i < g_manager.count - 1; i++) {
        g_manager.modules[i] = g_manager.modules[i + 1];
    }
    g_manager.count--;
    
    /* 清空最后一个位置 */
    memset(&g_manager.modules[g_manager.count], 0, sizeof(module_entry_t));
    
    return EDR_OK;
}

const edr_module_ops_t* edr_module_get(const char* name) {
    if (!g_manager.initialized) {
        return NULL;
    }
    
    int index = find_module_index(name);
    if (index < 0) {
        return NULL;
    }
    
    return g_manager.modules[index].ops;
}

edr_error_t edr_module_list(edr_module_type_t type, const edr_module_ops_t** list, 
                            size_t max_count, size_t* actual_count) {
    if (!g_manager.initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    if (list == NULL || max_count == 0) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    size_t count = 0;
    for (size_t i = 0; i < g_manager.count && count < max_count; i++) {
        if (g_manager.modules[i].ops != NULL && 
            g_manager.modules[i].ops->type == type) {
            list[count++] = g_manager.modules[i].ops;
        }
    }
    
    if (actual_count != NULL) {
        *actual_count = count;
    }
    
    return EDR_OK;
}

edr_error_t edr_module_start_all(void* config) {
    if (!g_manager.initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    edr_error_t result = EDR_OK;
    
    /* 按注册顺序依次初始化和启动 */
    for (size_t i = 0; i < g_manager.count; i++) {
        module_entry_t* entry = &g_manager.modules[i];
        
        if (entry->ops == NULL) {
            continue;
        }
        
        /* 初始化 */
        if (!entry->initialized && entry->ops->init != NULL) {
            edr_error_t err = entry->ops->init(config);
            if (err != EDR_OK) {
                result = err;
                continue;  /* 继续尝试启动其他模块 */
            }
            entry->initialized = true;
        }
        
        /* 启动 */
        if (!entry->running && entry->ops->start != NULL) {
            edr_error_t err = entry->ops->start();
            if (err != EDR_OK) {
                result = err;
                continue;
            }
            entry->running = true;
        }
    }
    
    return result;
}

edr_error_t edr_module_stop_all(void) {
    if (!g_manager.initialized) {
        return EDR_ERR_NOT_INITIALIZED;
    }
    
    edr_error_t result = EDR_OK;
    
    /* 按逆序停止 */
    for (size_t i = g_manager.count; i > 0; i--) {
        module_entry_t* entry = &g_manager.modules[i - 1];
        
        if (entry->ops == NULL) {
            continue;
        }
        
        /* 停止 */
        if (entry->running && entry->ops->stop != NULL) {
            edr_error_t err = entry->ops->stop();
            if (err != EDR_OK) {
                result = err;
            }
            entry->running = false;
        }
    }
    
    return result;
}

size_t edr_module_count(void) {
    if (!g_manager.initialized) {
        return 0;
    }
    return g_manager.count;
}
