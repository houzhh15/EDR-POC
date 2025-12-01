/**
 * @file pal_test.c
 * @brief Platform Abstraction Layer - 单元测试
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "pal.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>

/* ============================================================
 * 测试辅助宏
 * ============================================================ */

#define TEST_ASSERT(cond, msg) do { \
    if (!(cond)) { \
        printf("FAILED: %s - %s\n", __func__, msg); \
        return 1; \
    } \
} while(0)

#define TEST_PASS() do { \
    printf("PASSED: %s\n", __func__); \
    return 0; \
} while(0)

/* ============================================================
 * 测试用例
 * ============================================================ */

/* 测试 PAL 初始化 */
static int test_pal_init(void) {
    edr_error_t err = pal_init();
    TEST_ASSERT(err == EDR_OK, "pal_init should succeed");
    
    /* 重复初始化应返回错误 */
    err = pal_init();
    TEST_ASSERT(err == EDR_ERR_ALREADY_INITIALIZED, "double init should fail");
    
    pal_cleanup();
    
    /* 清理后可以重新初始化 */
    err = pal_init();
    TEST_ASSERT(err == EDR_OK, "reinit after cleanup should succeed");
    
    pal_cleanup();
    
    TEST_PASS();
}

/* 测试互斥锁创建和销毁 */
static int test_mutex_create_destroy(void) {
    pal_init();
    
    pal_mutex_t mutex = pal_mutex_create();
    TEST_ASSERT(mutex != NULL, "mutex create should succeed");
    
    pal_mutex_destroy(mutex);
    
    /* 销毁 NULL 不应崩溃 */
    pal_mutex_destroy(NULL);
    
    pal_cleanup();
    TEST_PASS();
}

/* 测试互斥锁加锁解锁 */
static int test_mutex_lock_unlock(void) {
    pal_init();
    
    pal_mutex_t mutex = pal_mutex_create();
    TEST_ASSERT(mutex != NULL, "mutex create should succeed");
    
    edr_error_t err = pal_mutex_lock(mutex);
    TEST_ASSERT(err == EDR_OK, "lock should succeed");
    
    err = pal_mutex_unlock(mutex);
    TEST_ASSERT(err == EDR_OK, "unlock should succeed");
    
    /* NULL 参数应返回错误 */
    err = pal_mutex_lock(NULL);
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "lock NULL should fail");
    
    err = pal_mutex_unlock(NULL);
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "unlock NULL should fail");
    
    pal_mutex_destroy(mutex);
    pal_cleanup();
    TEST_PASS();
}

/* 测试时间戳获取 */
static int test_time_now_ms(void) {
    pal_init();
    
    uint64_t t1 = pal_time_now_ms();
    TEST_ASSERT(t1 > 0, "timestamp should be positive");
    
    pal_sleep_ms(10);
    
    uint64_t t2 = pal_time_now_ms();
    TEST_ASSERT(t2 > t1, "timestamp should increase after sleep");
    TEST_ASSERT(t2 - t1 >= 10, "elapsed time should be at least 10ms");
    
    pal_cleanup();
    TEST_PASS();
}

/* 测试内存分配 */
static int test_mem_alloc(void) {
    pal_init();
    
    void* ptr = pal_mem_alloc(1024);
    TEST_ASSERT(ptr != NULL, "alloc should succeed");
    
    memset(ptr, 0xAB, 1024);
    pal_mem_free(ptr);
    
    /* 分配 0 字节应返回 NULL */
    ptr = pal_mem_alloc(0);
    TEST_ASSERT(ptr == NULL, "alloc 0 should return NULL");
    
    /* calloc 测试 */
    ptr = pal_mem_calloc(10, 100);
    TEST_ASSERT(ptr != NULL, "calloc should succeed");
    
    /* 验证清零 */
    unsigned char* bytes = (unsigned char*)ptr;
    for (int i = 0; i < 1000; i++) {
        TEST_ASSERT(bytes[i] == 0, "calloc should zero memory");
    }
    
    pal_mem_free(ptr);
    
    /* 释放 NULL 不应崩溃 */
    pal_mem_free(NULL);
    
    pal_cleanup();
    TEST_PASS();
}

/* 测试线程创建和等待 */
static void* thread_func(void* arg) {
    int* value = (int*)arg;
    *value = 42;
    return (void*)(intptr_t)(*value * 2);
}

static int test_thread_create_join(void) {
    pal_init();
    
    int value = 0;
    pal_thread_t thread = pal_thread_create(thread_func, &value);
    TEST_ASSERT(thread != NULL, "thread create should succeed");
    
    void* result = NULL;
    edr_error_t err = pal_thread_join(thread, &result);
    TEST_ASSERT(err == EDR_OK, "thread join should succeed");
    TEST_ASSERT(value == 42, "thread should modify value");
    TEST_ASSERT((intptr_t)result == 84, "thread should return correct value");
    
    pal_thread_destroy(thread);
    
    /* NULL 函数应返回 NULL */
    thread = pal_thread_create(NULL, NULL);
    TEST_ASSERT(thread == NULL, "thread create with NULL func should fail");
    
    pal_cleanup();
    TEST_PASS();
}

/* 测试文件存在检查 */
static int test_file_exists(void) {
    pal_init();
    
    /* /etc/passwd 在 macOS/Linux 上应存在 */
    bool exists = pal_file_exists("/etc/passwd");
    TEST_ASSERT(exists == true, "/etc/passwd should exist");
    
    /* 不存在的文件 */
    exists = pal_file_exists("/nonexistent/file/path");
    TEST_ASSERT(exists == false, "nonexistent file should not exist");
    
    /* NULL 路径 */
    exists = pal_file_exists(NULL);
    TEST_ASSERT(exists == false, "NULL path should return false");
    
    pal_cleanup();
    TEST_PASS();
}

/* ============================================================
 * 测试入口
 * ============================================================ */

int main(void) {
    int failures = 0;
    
    printf("=== PAL Unit Tests ===\n\n");
    
    failures += test_pal_init();
    failures += test_mutex_create_destroy();
    failures += test_mutex_lock_unlock();
    failures += test_time_now_ms();
    failures += test_mem_alloc();
    failures += test_thread_create_join();
    failures += test_file_exists();
    
    printf("\n=== Results: %d failures ===\n", failures);
    
    return failures > 0 ? 1 : 0;
}
