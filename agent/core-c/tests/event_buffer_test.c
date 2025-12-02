/**
 * @file event_buffer_test.c
 * @brief 事件缓冲队列(Event Buffer)单元测试
 * 
 * 测试SPSC无锁环形缓冲区的push/pop操作、并发性、边界条件等
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#include "event_buffer.h"
#include "edr_errors.h"
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <string.h>

#ifdef _WIN32
#include <windows.h>
#include <process.h>
#define THREAD_HANDLE HANDLE
#define CREATE_THREAD(func, arg) (HANDLE)_beginthreadex(NULL, 0, func, arg, 0, NULL)
#define JOIN_THREAD(handle) WaitForSingleObject(handle, INFINITE)
#define SLEEP_MS(ms) Sleep(ms)
#else
#include <pthread.h>
#include <unistd.h>
#define THREAD_HANDLE pthread_t
#define CREATE_THREAD(func, arg) ({ pthread_t t; pthread_create(&t, NULL, (void*(*)(void*))func, arg); t; })
#define JOIN_THREAD(handle) pthread_join(handle, NULL)
#define SLEEP_MS(ms) usleep((ms)*1000)
#endif

/**
 * @brief 测试1: Buffer创建和销毁
 */
void test_event_buffer_create_destroy() {
    printf("Test 1: Event Buffer Create/Destroy\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    assert(buffer->write_pos == 0);
    assert(buffer->read_pos == 0);
    assert(buffer->total_pushed == 0);
    assert(buffer->total_popped == 0);
    assert(buffer->dropped_count == 0);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Create/destroy successful\n");
}

/**
 * @brief 测试2: 单个事件Push/Pop
 */
void test_event_buffer_push_pop_single() {
    printf("Test 2: Event Buffer Push/Pop Single\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    // 创建测试事件
    edr_process_event_t event = {0};
    event.timestamp = 123456789;
    event.pid = 1234;
    event.ppid = 567;
    event.event_type = EDR_PROCESS_START;
    strcpy_s(event.process_name, sizeof(event.process_name), "test.exe");
    strcpy_s(event.executable_path, sizeof(event.executable_path), "C:\\test\\test.exe");
    
    // Push事件
    int result = event_buffer_push(buffer, &event);
    assert(result == EDR_SUCCESS);
    assert(buffer->total_pushed == 1);
    
    // Pop事件
    edr_process_event_t popped = {0};
    result = event_buffer_pop(buffer, &popped);
    assert(result == EDR_SUCCESS);
    assert(buffer->total_popped == 1);
    
    // 验证数据完整性
    assert(popped.timestamp == event.timestamp);
    assert(popped.pid == event.pid);
    assert(popped.ppid == event.ppid);
    assert(popped.event_type == event.event_type);
    assert(strcmp(popped.process_name, event.process_name) == 0);
    assert(strcmp(popped.executable_path, event.executable_path) == 0);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Single push/pop successful\n");
}

/**
 * @brief 测试3: 批量Push/Pop
 */
void test_event_buffer_push_pop_batch() {
    printf("Test 3: Event Buffer Batch Push/Pop\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    const int batch_size = 100;
    edr_process_event_t events[batch_size];
    
    // 准备批量事件
    for (int i = 0; i < batch_size; i++) {
        memset(&events[i], 0, sizeof(edr_process_event_t));
        events[i].timestamp = i;
        events[i].pid = 1000 + i;
        events[i].event_type = EDR_PROCESS_START;
    }
    
    // 批量Push
    for (int i = 0; i < batch_size; i++) {
        int result = event_buffer_push(buffer, &events[i]);
        assert(result == EDR_SUCCESS);
    }
    
    assert(buffer->total_pushed == batch_size);
    
    // 批量Pop
    edr_process_event_t popped_events[batch_size];
    int popped_count = event_buffer_pop_batch(buffer, popped_events, batch_size);
    assert(popped_count == batch_size);
    assert(buffer->total_popped == batch_size);
    
    // 验证数据
    for (int i = 0; i < batch_size; i++) {
        assert(popped_events[i].timestamp == (uint64_t)i);
        assert(popped_events[i].pid == (uint32_t)(1000 + i));
    }
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Batch push/pop successful\n");
}

/**
 * @brief 测试4: Buffer满时丢弃事件
 */
void test_event_buffer_full() {
    printf("Test 4: Event Buffer Full (Drop Events)\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    edr_process_event_t event = {0};
    event.pid = 1234;
    
    // 填满buffer (容量4096)
    int success_count = 0;
    for (int i = 0; i < EVENT_BUFFER_SIZE + 100; i++) {
        event.timestamp = i;
        int result = event_buffer_push(buffer, &event);
        if (result == EDR_SUCCESS) {
            success_count++;
        }
    }
    
    // 应该只能push 4095个(留一个空位区分满和空)
    assert(success_count == EVENT_BUFFER_SIZE - 1);
    assert(buffer->dropped_count > 0);
    printf("  Dropped events: %llu\n", (unsigned long long)buffer->dropped_count);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Buffer full handling correct\n");
}

/**
 * @brief 测试5: Buffer空时Pop
 */
void test_event_buffer_empty() {
    printf("Test 5: Event Buffer Empty Pop\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    edr_process_event_t event = {0};
    
    // 从空buffer pop
    int result = event_buffer_pop(buffer, &event);
    assert(result == EDR_ERROR_BUFFER_EMPTY);
    
    // 批量pop
    edr_process_event_t events[10];
    int count = event_buffer_pop_batch(buffer, events, 10);
    assert(count == 0);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Empty buffer handling correct\n");
}

/**
 * @brief 生产者线程函数
 */
#ifdef _WIN32
unsigned int __stdcall producer_thread(void* arg) {
#else
void* producer_thread(void* arg) {
#endif
    event_buffer_t* buffer = (event_buffer_t*)arg;
    
    for (int i = 0; i < 1000; i++) {
        edr_process_event_t event = {0};
        event.timestamp = i;
        event.pid = 2000 + i;
        
        event_buffer_push(buffer, &event);
        
        // 随机延迟
        if (i % 100 == 0) {
            SLEEP_MS(1);
        }
    }
    
#ifdef _WIN32
    return 0;
#else
    return NULL;
#endif
}

/**
 * @brief 消费者线程函数
 */
#ifdef _WIN32
unsigned int __stdcall consumer_thread(void* arg) {
#else
void* consumer_thread(void* arg) {
#endif
    event_buffer_t* buffer = (event_buffer_t*)arg;
    int total_consumed = 0;
    
    while (total_consumed < 1000) {
        edr_process_event_t events[100];
        int count = event_buffer_pop_batch(buffer, events, 100);
        total_consumed += count;
        
        if (count == 0) {
            SLEEP_MS(1);
        }
    }
    
#ifdef _WIN32
    return 0;
#else
    return NULL;
#endif
}

/**
 * @brief 测试6: 并发Push/Pop
 */
void test_event_buffer_concurrent() {
    printf("Test 6: Event Buffer Concurrent Push/Pop\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    // 创建生产者和消费者线程
    THREAD_HANDLE prod_thread = CREATE_THREAD(producer_thread, buffer);
    THREAD_HANDLE cons_thread = CREATE_THREAD(consumer_thread, buffer);
    
    // 等待完成
    JOIN_THREAD(prod_thread);
    JOIN_THREAD(cons_thread);
    
    // 验证统计
    printf("  Total pushed: %llu\n", (unsigned long long)buffer->total_pushed);
    printf("  Total popped: %llu\n", (unsigned long long)buffer->total_popped);
    printf("  Dropped: %llu\n", (unsigned long long)buffer->dropped_count);
    
    assert(buffer->total_popped == 1000);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Concurrent operations successful\n");
}

/**
 * @brief 测试7: 统计信息
 */
void test_event_buffer_stats() {
    printf("Test 7: Event Buffer Statistics\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    // Push一些事件
    edr_process_event_t event = {0};
    for (int i = 0; i < 50; i++) {
        event_buffer_push(buffer, &event);
    }
    
    // Pop一些事件
    for (int i = 0; i < 30; i++) {
        event_buffer_pop(buffer, &event);
    }
    
    // 检查统计
    assert(buffer->total_pushed == 50);
    assert(buffer->total_popped == 30);
    
    // 计算当前使用量: (write_pos - read_pos + SIZE) % SIZE
    uint32_t usage = (buffer->write_pos - buffer->read_pos + EVENT_BUFFER_SIZE) % EVENT_BUFFER_SIZE;
    printf("  Current usage: %u / %u\n", usage, EVENT_BUFFER_SIZE);
    assert(usage == 20);
    
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Statistics tracking correct\n");
}

/**
 * @brief 主测试入口
 */
int main(void) {
    printf("======================================\n");
    printf("Event Buffer (SPSC Ring Buffer) Unit Tests\n");
    printf("======================================\n\n");
    
    test_event_buffer_create_destroy();
    test_event_buffer_push_pop_single();
    test_event_buffer_push_pop_batch();
    test_event_buffer_full();
    test_event_buffer_empty();
    test_event_buffer_concurrent();
    test_event_buffer_stats();
    
    printf("\n======================================\n");
    printf("All tests passed!\n");
    printf("======================================\n");
    
    return 0;
}
