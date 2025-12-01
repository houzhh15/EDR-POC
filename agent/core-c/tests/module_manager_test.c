/**
 * @file module_manager_test.c
 * @brief 模块管理器单元测试
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "plugin/module_manager.h"
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
 * 模拟模块实现
 * ============================================================ */

static int g_mock_init_count = 0;
static int g_mock_start_count = 0;
static int g_mock_stop_count = 0;
static int g_mock_cleanup_count = 0;

static void reset_mock_counters(void) {
    g_mock_init_count = 0;
    g_mock_start_count = 0;
    g_mock_stop_count = 0;
    g_mock_cleanup_count = 0;
}

static edr_error_t mock_init(void* config) {
    (void)config;
    g_mock_init_count++;
    return EDR_OK;
}

static edr_error_t mock_start(void) {
    g_mock_start_count++;
    return EDR_OK;
}

static edr_error_t mock_stop(void) {
    g_mock_stop_count++;
    return EDR_OK;
}

static void mock_cleanup(void) {
    g_mock_cleanup_count++;
}

static edr_error_t mock_init_fail(void* config) {
    (void)config;
    return EDR_ERR_UNKNOWN;
}

/* 模拟模块定义 */
static const edr_module_ops_t mock_collector = {
    .name = "mock_collector",
    .version = "1.0.0",
    .type = EDR_MODULE_COLLECTOR,
    .init = mock_init,
    .start = mock_start,
    .stop = mock_stop,
    .cleanup = mock_cleanup,
};

static const edr_module_ops_t mock_detector = {
    .name = "mock_detector",
    .version = "1.0.0",
    .type = EDR_MODULE_DETECTOR,
    .init = mock_init,
    .start = mock_start,
    .stop = mock_stop,
    .cleanup = mock_cleanup,
};

static const edr_module_ops_t mock_responder = {
    .name = "mock_responder",
    .version = "1.0.0",
    .type = EDR_MODULE_RESPONDER,
    .init = mock_init,
    .start = mock_start,
    .stop = mock_stop,
    .cleanup = mock_cleanup,
};

static const edr_module_ops_t mock_fail_module = {
    .name = "mock_fail",
    .version = "1.0.0",
    .type = EDR_MODULE_COLLECTOR,
    .init = mock_init_fail,
    .start = mock_start,
    .stop = mock_stop,
    .cleanup = mock_cleanup,
};

/* ============================================================
 * 测试用例
 * ============================================================ */

/* 测试模块管理器初始化 */
static int test_manager_init(void) {
    edr_error_t err = edr_module_manager_init();
    TEST_ASSERT(err == EDR_OK, "init should succeed");
    
    /* 重复初始化应失败 */
    err = edr_module_manager_init();
    TEST_ASSERT(err == EDR_ERR_ALREADY_INITIALIZED, "double init should fail");
    
    edr_module_manager_cleanup();
    
    /* 清理后可重新初始化 */
    err = edr_module_manager_init();
    TEST_ASSERT(err == EDR_OK, "reinit after cleanup should succeed");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试模块注册 */
static int test_register(void) {
    reset_mock_counters();
    edr_module_manager_init();
    
    edr_error_t err = edr_module_register(&mock_collector);
    TEST_ASSERT(err == EDR_OK, "register should succeed");
    TEST_ASSERT(edr_module_count() == 1, "count should be 1");
    
    err = edr_module_register(&mock_detector);
    TEST_ASSERT(err == EDR_OK, "register second should succeed");
    TEST_ASSERT(edr_module_count() == 2, "count should be 2");
    
    /* 注册同名模块应失败 */
    err = edr_module_register(&mock_collector);
    TEST_ASSERT(err == EDR_ERR_ALREADY_INITIALIZED, "duplicate name should fail");
    
    /* NULL 参数应失败 */
    err = edr_module_register(NULL);
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "NULL ops should fail");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试模块获取 */
static int test_get(void) {
    edr_module_manager_init();
    edr_module_register(&mock_collector);
    edr_module_register(&mock_detector);
    
    const edr_module_ops_t* ops = edr_module_get("mock_collector");
    TEST_ASSERT(ops != NULL, "get should succeed");
    TEST_ASSERT(ops == &mock_collector, "should return correct ops");
    
    ops = edr_module_get("mock_detector");
    TEST_ASSERT(ops != NULL, "get second should succeed");
    TEST_ASSERT(ops == &mock_detector, "should return correct ops");
    
    ops = edr_module_get("nonexistent");
    TEST_ASSERT(ops == NULL, "get nonexistent should return NULL");
    
    ops = edr_module_get(NULL);
    TEST_ASSERT(ops == NULL, "get NULL should return NULL");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试模块列表 */
static int test_list(void) {
    edr_module_manager_init();
    edr_module_register(&mock_collector);
    edr_module_register(&mock_detector);
    edr_module_register(&mock_responder);
    
    const edr_module_ops_t* list[10];
    size_t count = 0;
    
    edr_error_t err = edr_module_list(EDR_MODULE_COLLECTOR, list, 10, &count);
    TEST_ASSERT(err == EDR_OK, "list should succeed");
    TEST_ASSERT(count == 1, "should have 1 collector");
    TEST_ASSERT(list[0] == &mock_collector, "should be mock_collector");
    
    err = edr_module_list(EDR_MODULE_DETECTOR, list, 10, &count);
    TEST_ASSERT(err == EDR_OK, "list should succeed");
    TEST_ASSERT(count == 1, "should have 1 detector");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试启动和停止所有模块 */
static int test_start_stop_all(void) {
    reset_mock_counters();
    edr_module_manager_init();
    edr_module_register(&mock_collector);
    edr_module_register(&mock_detector);
    
    edr_error_t err = edr_module_start_all(NULL);
    TEST_ASSERT(err == EDR_OK, "start_all should succeed");
    TEST_ASSERT(g_mock_init_count == 2, "init should be called twice");
    TEST_ASSERT(g_mock_start_count == 2, "start should be called twice");
    
    err = edr_module_stop_all();
    TEST_ASSERT(err == EDR_OK, "stop_all should succeed");
    TEST_ASSERT(g_mock_stop_count == 2, "stop should be called twice");
    
    edr_module_manager_cleanup();
    TEST_ASSERT(g_mock_cleanup_count == 2, "cleanup should be called twice");
    
    TEST_PASS();
}

/* 测试模块注销 */
static int test_unregister(void) {
    reset_mock_counters();
    edr_module_manager_init();
    edr_module_register(&mock_collector);
    edr_module_register(&mock_detector);
    edr_module_register(&mock_responder);
    
    TEST_ASSERT(edr_module_count() == 3, "should have 3 modules");
    
    /* 启动后注销 */
    edr_module_start_all(NULL);
    
    edr_error_t err = edr_module_unregister("mock_detector");
    TEST_ASSERT(err == EDR_OK, "unregister should succeed");
    TEST_ASSERT(edr_module_count() == 2, "should have 2 modules");
    TEST_ASSERT(g_mock_stop_count == 1, "stop should be called once");
    TEST_ASSERT(g_mock_cleanup_count == 1, "cleanup should be called once");
    
    /* 验证已删除 */
    const edr_module_ops_t* ops = edr_module_get("mock_detector");
    TEST_ASSERT(ops == NULL, "unregistered module should not exist");
    
    /* 注销不存在的模块 */
    err = edr_module_unregister("nonexistent");
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "unregister nonexistent should fail");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试模块初始化失败 */
static int test_init_failure(void) {
    reset_mock_counters();
    edr_module_manager_init();
    edr_module_register(&mock_fail_module);
    edr_module_register(&mock_collector);
    
    edr_error_t err = edr_module_start_all(NULL);
    /* 即使有模块失败，也应该继续尝试启动其他模块 */
    TEST_ASSERT(err != EDR_OK, "start_all should return error");
    TEST_ASSERT(g_mock_init_count == 1, "successful init should be called");
    TEST_ASSERT(g_mock_start_count == 1, "successful start should be called");
    
    edr_module_manager_cleanup();
    TEST_PASS();
}

/* 测试未初始化调用 */
static int test_not_initialized(void) {
    /* 不初始化就调用各函数 */
    edr_error_t err = edr_module_register(&mock_collector);
    TEST_ASSERT(err == EDR_ERR_NOT_INITIALIZED, "register without init should fail");
    
    err = edr_module_unregister("test");
    TEST_ASSERT(err == EDR_ERR_NOT_INITIALIZED, "unregister without init should fail");
    
    const edr_module_ops_t* ops = edr_module_get("test");
    TEST_ASSERT(ops == NULL, "get without init should return NULL");
    
    TEST_ASSERT(edr_module_count() == 0, "count without init should be 0");
    
    TEST_PASS();
}

/* ============================================================
 * 测试入口
 * ============================================================ */

int main(void) {
    int failures = 0;
    
    printf("=== Module Manager Unit Tests ===\n\n");
    
    failures += test_manager_init();
    failures += test_register();
    failures += test_get();
    failures += test_list();
    failures += test_start_stop_all();
    failures += test_unregister();
    failures += test_init_failure();
    failures += test_not_initialized();
    
    printf("\n=== Results: %d failures ===\n", failures);
    
    return failures > 0 ? 1 : 0;
}
