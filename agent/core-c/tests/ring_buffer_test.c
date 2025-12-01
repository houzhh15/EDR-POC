/**
 * @file ring_buffer_test.c
 * @brief SPSC 环形缓冲队列单元测试
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "queue/ring_buffer.h"
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

/* 测试队列创建 */
static int test_create(void) {
    /* 正常创建 (容量是 2 的幂) */
    edr_ring_buffer_t* rb = ring_buffer_create(16);
    TEST_ASSERT(rb != NULL, "create with power of 2 should succeed");
    TEST_ASSERT(rb->capacity == 16, "capacity should be 16");
    TEST_ASSERT(rb->mask == 15, "mask should be 15");
    ring_buffer_destroy(rb);
    
    /* 容量不是 2 的幂应失败 */
    rb = ring_buffer_create(10);
    TEST_ASSERT(rb == NULL, "create with non-power of 2 should fail");
    
    rb = ring_buffer_create(0);
    TEST_ASSERT(rb == NULL, "create with 0 should fail");
    
    TEST_PASS();
}

/* 测试空队列 */
static int test_empty_queue(void) {
    edr_ring_buffer_t* rb = ring_buffer_create(8);
    TEST_ASSERT(rb != NULL, "create should succeed");
    
    TEST_ASSERT(ring_buffer_is_empty(rb) == true, "new queue should be empty");
    TEST_ASSERT(ring_buffer_is_full(rb) == false, "new queue should not be full");
    TEST_ASSERT(ring_buffer_size(rb) == 0, "new queue size should be 0");
    
    edr_event_t* event = ring_buffer_pop(rb);
    TEST_ASSERT(event == NULL, "pop from empty queue should return NULL");
    
    ring_buffer_destroy(rb);
    TEST_PASS();
}

/* 测试事件创建和销毁 */
static int test_event_create_destroy(void) {
    const char* data = "{\"test\":\"data\"}";
    edr_event_t* event = edr_event_create(1, 1234567890, data, strlen(data));
    TEST_ASSERT(event != NULL, "event create should succeed");
    TEST_ASSERT(event->type == 1, "type should match");
    TEST_ASSERT(event->timestamp == 1234567890, "timestamp should match");
    TEST_ASSERT(event->data_len == strlen(data), "data_len should match");
    TEST_ASSERT(strcmp(event->data, data) == 0, "data should match");
    
    edr_event_destroy(event);
    
    /* 空数据 */
    event = edr_event_create(2, 0, NULL, 0);
    TEST_ASSERT(event != NULL, "event create with NULL data should succeed");
    TEST_ASSERT(event->data_len == 0, "data_len should be 0");
    edr_event_destroy(event);
    
    /* 无效参数 */
    event = edr_event_create(1, 0, NULL, 100);
    TEST_ASSERT(event == NULL, "event create with NULL data but non-zero len should fail");
    
    TEST_PASS();
}

/* 测试正常入队出队 */
static int test_push_pop(void) {
    edr_ring_buffer_t* rb = ring_buffer_create(8);
    TEST_ASSERT(rb != NULL, "create should succeed");
    
    /* 入队 3 个事件 */
    for (int i = 0; i < 3; i++) {
        char data[32];
        snprintf(data, sizeof(data), "{\"id\":%d}", i);
        edr_event_t* event = edr_event_create(i, i * 1000, data, strlen(data));
        TEST_ASSERT(event != NULL, "event create should succeed");
        
        edr_error_t err = ring_buffer_push(rb, event);
        TEST_ASSERT(err == EDR_OK, "push should succeed");
    }
    
    TEST_ASSERT(ring_buffer_size(rb) == 3, "size should be 3");
    TEST_ASSERT(ring_buffer_is_empty(rb) == false, "queue should not be empty");
    
    /* 出队验证 */
    for (int i = 0; i < 3; i++) {
        edr_event_t* event = ring_buffer_pop(rb);
        TEST_ASSERT(event != NULL, "pop should succeed");
        TEST_ASSERT(event->type == (uint32_t)i, "type should match");
        TEST_ASSERT(event->timestamp == (uint64_t)(i * 1000), "timestamp should match");
        edr_event_destroy(event);
    }
    
    TEST_ASSERT(ring_buffer_is_empty(rb) == true, "queue should be empty after pop all");
    TEST_ASSERT(ring_buffer_size(rb) == 0, "size should be 0");
    
    ring_buffer_destroy(rb);
    TEST_PASS();
}

/* 测试队列满 */
static int test_queue_full(void) {
    edr_ring_buffer_t* rb = ring_buffer_create(4);
    TEST_ASSERT(rb != NULL, "create should succeed");
    
    /* 填满队列 */
    for (int i = 0; i < 4; i++) {
        edr_event_t* event = edr_event_create(i, 0, "x", 1);
        edr_error_t err = ring_buffer_push(rb, event);
        TEST_ASSERT(err == EDR_OK, "push should succeed");
    }
    
    TEST_ASSERT(ring_buffer_is_full(rb) == true, "queue should be full");
    TEST_ASSERT(ring_buffer_size(rb) == 4, "size should be 4");
    
    /* 队满再入队应失败 */
    edr_event_t* extra = edr_event_create(99, 0, "x", 1);
    edr_error_t err = ring_buffer_push(rb, extra);
    TEST_ASSERT(err == EDR_ERR_NO_MEMORY, "push to full queue should fail");
    edr_event_destroy(extra);  /* 入队失败，需要手动释放 */
    
    ring_buffer_destroy(rb);
    TEST_PASS();
}

/* 测试环绕 (wrap around) */
static int test_wrap_around(void) {
    edr_ring_buffer_t* rb = ring_buffer_create(4);
    TEST_ASSERT(rb != NULL, "create should succeed");
    
    /* 入队出队多轮，测试索引环绕 */
    for (int round = 0; round < 10; round++) {
        /* 入队 3 个 */
        for (int i = 0; i < 3; i++) {
            edr_event_t* event = edr_event_create(round * 10 + i, 0, "x", 1);
            edr_error_t err = ring_buffer_push(rb, event);
            TEST_ASSERT(err == EDR_OK, "push should succeed");
        }
        
        /* 出队 3 个 */
        for (int i = 0; i < 3; i++) {
            edr_event_t* event = ring_buffer_pop(rb);
            TEST_ASSERT(event != NULL, "pop should succeed");
            TEST_ASSERT(event->type == (uint32_t)(round * 10 + i), "type should match");
            edr_event_destroy(event);
        }
    }
    
    TEST_ASSERT(ring_buffer_is_empty(rb) == true, "queue should be empty");
    
    ring_buffer_destroy(rb);
    TEST_PASS();
}

/* 测试 NULL 参数 */
static int test_null_params(void) {
    TEST_ASSERT(ring_buffer_is_full(NULL) == true, "is_full(NULL) should return true");
    TEST_ASSERT(ring_buffer_is_empty(NULL) == true, "is_empty(NULL) should return true");
    TEST_ASSERT(ring_buffer_size(NULL) == 0, "size(NULL) should return 0");
    TEST_ASSERT(ring_buffer_pop(NULL) == NULL, "pop(NULL) should return NULL");
    
    edr_ring_buffer_t* rb = ring_buffer_create(4);
    edr_error_t err = ring_buffer_push(rb, NULL);
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "push(rb, NULL) should fail");
    
    err = ring_buffer_push(NULL, NULL);
    TEST_ASSERT(err == EDR_ERR_INVALID_PARAM, "push(NULL, NULL) should fail");
    
    ring_buffer_destroy(rb);
    ring_buffer_destroy(NULL);  /* 应不崩溃 */
    
    TEST_PASS();
}

/* ============================================================
 * 测试入口
 * ============================================================ */

int main(void) {
    int failures = 0;
    
    printf("=== Ring Buffer Unit Tests ===\n\n");
    
    failures += test_create();
    failures += test_empty_queue();
    failures += test_event_create_destroy();
    failures += test_push_pop();
    failures += test_queue_full();
    failures += test_wrap_around();
    failures += test_null_params();
    
    printf("\n=== Results: %d failures ===\n", failures);
    
    return failures > 0 ? 1 : 0;
}
