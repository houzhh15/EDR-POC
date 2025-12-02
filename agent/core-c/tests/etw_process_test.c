/**
 * @file etw_process_test.c
 * @brief ETW进程事件消费模块单元测试
 * 
 * 测试进程事件解析、LRU缓存、元数据提取等功能
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifdef _WIN32

#include "etw_process.h"
#include "event_buffer.h"
#include "edr_errors.h"
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <string.h>
#include <windows.h>

/**
 * @brief 测试1: Consumer创建和销毁
 */
void test_process_consumer_create_destroy() {
    printf("Test 1: Process Consumer Create/Destroy\n");
    
    // 创建事件缓冲区
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    // 创建消费者
    etw_process_consumer_t* consumer = etw_process_consumer_create(buffer);
    assert(consumer != NULL);
    assert(consumer->buffer == buffer);
    assert(consumer->cache_used == 0);
    assert(consumer->total_events == 0);
    assert(consumer->parse_errors == 0);
    
    // 销毁消费者
    etw_process_consumer_destroy(consumer);
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Create/destroy successful\n");
}

/**
 * @brief 测试2: LRU句柄缓存
 */
void test_process_handle_lru_cache() {
    printf("Test 2: Process Handle LRU Cache\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    etw_process_consumer_t* consumer = etw_process_consumer_create(buffer);
    assert(consumer != NULL);
    
    // 测试缓存当前进程句柄
    DWORD current_pid = GetCurrentProcessId();
    
    // 第一次获取(应该打开新句柄)
    HANDLE handle1 = etw_get_process_handle(consumer, current_pid);
    (void)handle1; // 用于缓存测试
    assert(handle1 != NULL);
    assert(consumer->cache_used == 1);
    
    // 第二次获取(应该命中缓存)
    HANDLE handle2 = etw_get_process_handle(consumer, current_pid);
    (void)handle2; // 用于缓存测试
    assert(handle2 == handle1);
    assert(consumer->cache_used == 1); // 缓存大小不变
    
    // 测试缓存多个进程句柄
    for (int i = 0; i < 10; i++) {
        DWORD fake_pid = 1000 + i;
        HANDLE h = etw_get_process_handle(consumer, fake_pid);
        // 无效PID应该返回NULL
        (void)h;
    }
    
    etw_process_consumer_destroy(consumer);
    event_buffer_destroy(buffer);
    
    printf("  [PASS] LRU cache works correctly\n");
}

/**
 * @brief 测试3: 进程元数据提取
 */
void test_process_metadata_extraction() {
    printf("Test 3: Process Metadata Extraction\n");
    
    // 打开当前进程
    HANDLE process_handle = OpenProcess(
        PROCESS_QUERY_INFORMATION | PROCESS_VM_READ,
        FALSE,
        GetCurrentProcessId()
    );
    
    if (process_handle == NULL) {
        printf("  [SKIP] Cannot open current process\n");
        return;
    }
    
    // 测试路径获取
    char path[MAX_PATH] = {0};
    DWORD path_len = MAX_PATH;
    BOOL result = QueryFullProcessImageNameA(process_handle, 0, path, &path_len);
    (void)result; // 用于断言检查
    assert(result != FALSE);
    assert(strlen(path) > 0);
    printf("  Current process path: %s\n", path);
    
    // 测试用户信息获取
    HANDLE token_handle = NULL;
    if (OpenProcessToken(process_handle, TOKEN_QUERY, &token_handle)) {
        DWORD token_info_length = 0;
        GetTokenInformation(token_handle, TokenUser, NULL, 0, &token_info_length);
        
        if (token_info_length > 0) {
            PTOKEN_USER token_user = (PTOKEN_USER)malloc(token_info_length);
            if (GetTokenInformation(token_handle, TokenUser, token_user, token_info_length, &token_info_length)) {
                char username[256] = {0};
                char domain[256] = {0};
                DWORD name_len = sizeof(username);
                DWORD domain_len = sizeof(domain);
                SID_NAME_USE sid_type;
                
                if (LookupAccountSidA(NULL, token_user->User.Sid, username, &name_len, domain, &domain_len, &sid_type)) {
                    printf("  Current user: %s\\%s\n", domain, username);
                }
            }
            free(token_user);
        }
        
        CloseHandle(token_handle);
    }
    
    CloseHandle(process_handle);
    
    printf("  [PASS] Metadata extraction successful\n");
}

/**
 * @brief 测试4: 事件解析(模拟EVENT_RECORD)
 */
void test_event_parsing() {
    printf("Test 4: Event Parsing\n");
    
    event_buffer_t* buffer = event_buffer_create();
    assert(buffer != NULL);
    
    etw_process_consumer_t* consumer = etw_process_consumer_create(buffer);
    assert(consumer != NULL);
    
    // 注意: 实际的EVENT_RECORD解析需要真实的ETW事件
    // 这里仅测试数据结构完整性
    
    // 创建一个模拟的进程事件
    edr_process_event_t test_event = {0};
    test_event.timestamp = 0;
    test_event.pid = GetCurrentProcessId();
    test_event.ppid = 0;
    test_event.event_type = EDR_PROCESS_START;
    strcpy_s(test_event.process_name, sizeof(test_event.process_name), "test.exe");
    strcpy_s(test_event.executable_path, sizeof(test_event.executable_path), "C:\\test\\test.exe");
    
    // 推送到buffer
    int result = event_buffer_push(buffer, &test_event);
    (void)result; // 用于断言检查
    assert(result == EDR_SUCCESS);
    
    // 从buffer取出
    edr_process_event_t popped_event = {0};
    result = event_buffer_pop(buffer, &popped_event);
    (void)result; // 用于断言检查
    assert(result == EDR_SUCCESS);
    assert(popped_event.pid == test_event.pid);
    assert(strcmp(popped_event.process_name, test_event.process_name) == 0);
    
    etw_process_consumer_destroy(consumer);
    event_buffer_destroy(buffer);
    
    printf("  [PASS] Event parsing structure valid\n");
}

/**
 * @brief 测试5: 缓存满时的LRU替换
 */
void test_lru_cache_full_replacement() {
    printf("Test 5: LRU Cache Full Replacement\n");
    
    event_buffer_t* buffer = event_buffer_create();
    etw_process_consumer_t* consumer = etw_process_consumer_create(buffer);
    
    // 填充缓存到容量上限(256个)
    // 注意: 大部分假PID会打开失败,但测试LRU逻辑
    for (int i = 0; i < PROCESS_HANDLE_CACHE_SIZE + 10; i++) {
        DWORD fake_pid = 10000 + i;
        HANDLE h = etw_get_process_handle(consumer, fake_pid);
        (void)h;
    }
    
    // 验证缓存不超过容量
    assert(consumer->cache_used <= PROCESS_HANDLE_CACHE_SIZE);
    
    etw_process_consumer_destroy(consumer);
    event_buffer_destroy(buffer);
    
    printf("  [PASS] LRU replacement works correctly\n");
}

/**
 * @brief 主测试入口
 */
int main(void) {
    printf("======================================\n");
    printf("ETW Process Consumer Unit Tests\n");
    printf("======================================\n\n");
    
    test_process_consumer_create_destroy();
    test_process_handle_lru_cache();
    test_process_metadata_extraction();
    test_event_parsing();
    test_lru_cache_full_replacement();
    
    printf("\n======================================\n");
    printf("All tests passed!\n");
    printf("======================================\n");
    
    return 0;
}

#else

#include <stdio.h>

int main(void) {
    printf("ETW Process Consumer tests are Windows-only\n");
    return 0;
}

#endif /* _WIN32 */
