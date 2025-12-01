/**
 * @file pal_linux.c
 * @brief Platform Abstraction Layer - Linux 占位实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "pal.h"

/* ============================================================
 * PAL 初始化/清理
 * ============================================================ */

edr_error_t pal_init(void) {
    return EDR_ERR_NOT_SUPPORTED;
}

void pal_cleanup(void) {
    /* 占位 */
}

/* ============================================================
 * 互斥锁实现
 * ============================================================ */

pal_mutex_t pal_mutex_create(void) {
    return NULL;
}

void pal_mutex_destroy(pal_mutex_t mutex) {
    (void)mutex;
}

edr_error_t pal_mutex_lock(pal_mutex_t mutex) {
    (void)mutex;
    return EDR_ERR_NOT_SUPPORTED;
}

edr_error_t pal_mutex_unlock(pal_mutex_t mutex) {
    (void)mutex;
    return EDR_ERR_NOT_SUPPORTED;
}

/* ============================================================
 * 线程实现
 * ============================================================ */

pal_thread_t pal_thread_create(pal_thread_func_t func, void* arg) {
    (void)func;
    (void)arg;
    return NULL;
}

edr_error_t pal_thread_join(pal_thread_t thread, void** result) {
    (void)thread;
    (void)result;
    return EDR_ERR_NOT_SUPPORTED;
}

void pal_thread_destroy(pal_thread_t thread) {
    (void)thread;
}

/* ============================================================
 * 内存管理实现
 * ============================================================ */

void* pal_mem_alloc(size_t size) {
    (void)size;
    return NULL;
}

void pal_mem_free(void* ptr) {
    (void)ptr;
}

void* pal_mem_calloc(size_t count, size_t size) {
    (void)count;
    (void)size;
    return NULL;
}

/* ============================================================
 * 时间实现
 * ============================================================ */

uint64_t pal_time_now_ms(void) {
    return 0;
}

void pal_sleep_ms(uint32_t ms) {
    (void)ms;
}

/* ============================================================
 * 文件操作实现
 * ============================================================ */

edr_error_t pal_file_read(const char* path, void* buf, size_t size, size_t* bytes_read) {
    (void)path;
    (void)buf;
    (void)size;
    (void)bytes_read;
    return EDR_ERR_NOT_SUPPORTED;
}

edr_error_t pal_file_move(const char* src, const char* dst) {
    (void)src;
    (void)dst;
    return EDR_ERR_NOT_SUPPORTED;
}

bool pal_file_exists(const char* path) {
    (void)path;
    return false;
}

/* ============================================================
 * 进程管理实现
 * ============================================================ */

edr_error_t pal_process_get_list(pal_process_info_t* list, size_t max_count, size_t* actual_count) {
    (void)list;
    (void)max_count;
    (void)actual_count;
    return EDR_ERR_NOT_SUPPORTED;
}

edr_error_t pal_process_terminate(uint32_t pid) {
    (void)pid;
    return EDR_ERR_NOT_SUPPORTED;
}
