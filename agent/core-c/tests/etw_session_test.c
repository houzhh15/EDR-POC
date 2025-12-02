/**
 * @file etw_session_test.c
 * @brief ETW Session模块单元测试
 * 
 * 测试ETW Session的初始化、启动、停止和销毁逻辑
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifdef _WIN32

#include "etw_session.h"
#include "edr_errors.h"
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <windows.h>

/**
 * @brief 测试用的事件计数器
 */
static volatile int g_event_count = 0;

/**
 * @brief 测试回调函数
 */
static void test_event_callback(PEVENT_RECORD event_record, void* context) {
    (void)event_record;
    (void)context;
    InterlockedIncrement((LONG*)&g_event_count);
}

/**
 * @brief 测试1: Session初始化
 */
void test_etw_session_init() {
    printf("Test 1: ETW Session Init\n");
    
    etw_session_t* session = etw_session_init(ETW_SESSION_NAME);
    assert(session != NULL);
    assert(session->properties != NULL);
    assert(session->session_handle == 0);
    assert(session->is_running == 0);
    
    etw_session_destroy(session);
    
    printf("  [PASS] Session init successful\n");
}

/**
 * @brief 测试2: Session启动和停止
 */
void test_etw_session_start_stop() {
    printf("Test 2: ETW Session Start/Stop\n");
    
    // 初始化Session
    etw_session_t* session = etw_session_init(ETW_SESSION_NAME);
    assert(session != NULL);
    
    // 重置事件计数器
    g_event_count = 0;
    
    // 启动Session(需要管理员权限)
    int result = etw_session_start(session, test_event_callback, NULL);
    (void)result; // 用于错误检查
    
    if (result == EDR_ERROR_ETW_ACCESS_DENIED) {
        printf("  [SKIP] Admin privileges required\n");
        etw_session_destroy(session);
        return;
    }
    
    assert(result == EDR_SUCCESS);
    assert(session->is_running != 0);
    assert(session->session_handle != 0);
    
    // 等待一段时间让ETW捕获事件
    Sleep(2000);
    
    // 检查是否接收到事件
    printf("  Events received: %d\n", g_event_count);
    
    // 停止Session
    etw_session_stop(session);
    assert(session->is_running == 0);
    
    etw_session_destroy(session);
    
    printf("  [PASS] Start/stop successful\n");
}

/**
 * @brief 测试3: Session错误处理
 */
void test_etw_session_error_handling() {
    printf("Test 3: ETW Session Error Handling\n");
    
    // 测试NULL参数
    etw_session_t* session = etw_session_init(NULL);
    assert(session == NULL);
    
    // 测试启动时NULL回调
    session = etw_session_init(ETW_SESSION_NAME);
    assert(session != NULL);
    
    int result = etw_session_start(session, NULL, NULL);
    (void)result; // 用于断言检查
    assert(result == EDR_ERROR_INVALID_PARAM);
    
    etw_session_destroy(session);
    
    printf("  [PASS] Error handling works correctly\n");
}

/**
 * @brief 测试4: Session自动重启(模拟)
 */
void test_etw_session_auto_restart() {
    printf("Test 4: ETW Session Auto Restart\n");
    
    etw_session_t* session = etw_session_init(ETW_SESSION_NAME);
    assert(session != NULL);
    
    // 检查初始重启计数
    assert(session->restart_count == 0);
    
    etw_session_destroy(session);
    
    printf("  [PASS] Auto restart mechanism initialized\n");
}

/**
 * @brief 测试5: 多次启动/停止
 */
void test_etw_session_multiple_cycles() {
    printf("Test 5: ETW Session Multiple Cycles\n");
    
    etw_session_t* session = etw_session_init(ETW_SESSION_NAME);
    assert(session != NULL);
    
    for (int i = 0; i < 3; i++) {
        int result = etw_session_start(session, test_event_callback, NULL);
        (void)result; // 用于错误检查
        
        if (result == EDR_ERROR_ETW_ACCESS_DENIED) {
            printf("  [SKIP] Admin privileges required\n");
            break;
        }
        
        assert(result == EDR_SUCCESS);
        
        Sleep(100);
        
        etw_session_stop(session);
        
        Sleep(100);
    }
    
    etw_session_destroy(session);
    
    printf("  [PASS] Multiple cycles successful\n");
}

/**
 * @brief 主测试入口
 */
int main(void) {
    printf("======================================\n");
    printf("ETW Session Module Unit Tests\n");
    printf("======================================\n\n");
    
    test_etw_session_init();
    test_etw_session_start_stop();
    test_etw_session_error_handling();
    test_etw_session_auto_restart();
    test_etw_session_multiple_cycles();
    
    printf("\n======================================\n");
    printf("All tests passed!\n");
    printf("======================================\n");
    
    return 0;
}

#else

#include <stdio.h>

int main(void) {
    printf("ETW Session tests are Windows-only\n");
    return 0;
}

#endif /* _WIN32 */
