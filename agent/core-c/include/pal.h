/**
 * @file pal.h
 * @brief Platform Abstraction Layer - 平台抽象层接口定义
 *
 * 此头文件定义了跨平台的统一接口，封装操作系统差异。
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#ifndef EDR_PAL_H
#define EDR_PAL_H

#ifdef __cplusplus
extern "C" {
#endif

#include "edr_core.h"
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

/* ============================================================
 * PAL 初始化/清理
 * ============================================================ */

/**
 * @brief 初始化平台抽象层
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_init(void);

/**
 * @brief 清理平台抽象层
 */
void pal_cleanup(void);

/* ============================================================
 * 互斥锁接口
 * ============================================================ */

/** 互斥锁句柄 (不透明类型) */
typedef struct pal_mutex* pal_mutex_t;

/**
 * @brief 创建互斥锁
 * @return 互斥锁句柄，失败返回 NULL
 */
pal_mutex_t pal_mutex_create(void);

/**
 * @brief 销毁互斥锁
 * @param mutex 互斥锁句柄
 */
void pal_mutex_destroy(pal_mutex_t mutex);

/**
 * @brief 加锁
 * @param mutex 互斥锁句柄
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_mutex_lock(pal_mutex_t mutex);

/**
 * @brief 解锁
 * @param mutex 互斥锁句柄
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_mutex_unlock(pal_mutex_t mutex);

/* ============================================================
 * 线程接口
 * ============================================================ */

/** 线程句柄 (不透明类型) */
typedef struct pal_thread* pal_thread_t;

/** 线程入口函数类型 */
typedef void* (*pal_thread_func_t)(void* arg);

/**
 * @brief 创建线程
 * @param func 线程入口函数
 * @param arg 传递给线程的参数
 * @return 线程句柄，失败返回 NULL
 */
pal_thread_t pal_thread_create(pal_thread_func_t func, void* arg);

/**
 * @brief 等待线程结束
 * @param thread 线程句柄
 * @param result 线程返回值 (可为 NULL)
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_thread_join(pal_thread_t thread, void** result);

/**
 * @brief 销毁线程句柄 (线程必须已结束)
 * @param thread 线程句柄
 */
void pal_thread_destroy(pal_thread_t thread);

/* ============================================================
 * 内存管理接口
 * ============================================================ */

/**
 * @brief 分配内存
 * @param size 字节数
 * @return 内存指针，失败返回 NULL
 */
void* pal_mem_alloc(size_t size);

/**
 * @brief 释放内存
 * @param ptr 内存指针
 */
void pal_mem_free(void* ptr);

/**
 * @brief 分配并清零内存
 * @param count 元素个数
 * @param size 每个元素大小
 * @return 内存指针，失败返回 NULL
 */
void* pal_mem_calloc(size_t count, size_t size);

/* ============================================================
 * 时间接口
 * ============================================================ */

/**
 * @brief 获取当前毫秒时间戳 (自 Epoch)
 * @return 毫秒时间戳
 */
uint64_t pal_time_now_ms(void);

/**
 * @brief 休眠指定毫秒
 * @param ms 毫秒数
 */
void pal_sleep_ms(uint32_t ms);

/* ============================================================
 * 文件操作接口
 * ============================================================ */

/**
 * @brief 读取文件内容
 * @param path 文件路径
 * @param buf 输出缓冲区
 * @param size 缓冲区大小
 * @param bytes_read 实际读取字节数 (可为 NULL)
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_file_read(const char* path, void* buf, size_t size, size_t* bytes_read);

/**
 * @brief 移动/重命名文件
 * @param src 源路径
 * @param dst 目标路径
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_file_move(const char* src, const char* dst);

/**
 * @brief 检查文件是否存在
 * @param path 文件路径
 * @return true 存在，false 不存在
 */
bool pal_file_exists(const char* path);

/* ============================================================
 * 进程管理接口
 * ============================================================ */

/** 进程信息结构 */
typedef struct pal_process_info {
    uint32_t pid;           /**< 进程 ID */
    uint32_t ppid;          /**< 父进程 ID */
    char name[256];         /**< 进程名称 */
    char path[1024];        /**< 可执行文件路径 */
} pal_process_info_t;

/**
 * @brief 获取进程列表
 * @param list 输出数组
 * @param max_count 数组最大容量
 * @param actual_count 实际进程数 (可为 NULL)
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_process_get_list(pal_process_info_t* list, size_t max_count, size_t* actual_count);

/**
 * @brief 终止进程
 * @param pid 进程 ID
 * @return EDR_OK 成功，其他值表示失败
 */
edr_error_t pal_process_terminate(uint32_t pid);

#ifdef __cplusplus
}
#endif

#endif /* EDR_PAL_H */
