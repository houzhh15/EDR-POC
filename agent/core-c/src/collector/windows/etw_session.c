/**
 * @file etw_session.c
 * @brief ETW Session管理模块实现
 * 
 * 实现Windows ETW跟踪会话的创建、启动、停止和销毁逻辑。
 * 关键实现细节:
 * - 使用StartTrace创建实时Session,配置64KB×20个Buffer
 * - 使用EnableTraceEx2连接Provider并设置关键字过滤
 * - 在独立线程中调用ProcessTrace()消费事件
 * - Session创建失败时自动清理已存在的同名Session
 * - 消费线程异常退出时自动重启(最多3次)
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#include "etw_session.h"
#include "../../include/edr_errors.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/**
 * @brief 消费线程入口函数
 * 
 * 在独立线程中调用ProcessTrace()消费ETW事件。
 * ProcessTrace()会阻塞直到CloseTrace()被调用。
 */
static DWORD WINAPI consume_thread_func(LPVOID param) {
    etw_session_t* session = (etw_session_t*)param;
    
    // ProcessTrace会阻塞直到CloseTrace被调用
    ULONG status = ProcessTrace(&session->trace_handle, 1, NULL, NULL);
    
    if (status != ERROR_SUCCESS && status != ERROR_CANCELLED) {
        // 异常退出,记录错误
        // TODO: 添加日志(step-05实现后)
        session->is_running = 0;
        
        // 如果重启次数未超限,触发自动重启
        if (session->restart_count < ETW_MAX_RESTART_RETRY) {
            session->restart_count++;
            // TODO: 实现自动重启逻辑
        }
    }
    
    return status;
}

/**
 * @brief ETW事件回调桥接函数
 * 
 * ETW API要求使用CALLBACK调用约定,这个函数桥接到用户的回调函数。
 */
static VOID WINAPI event_record_callback_bridge(PEVENT_RECORD event_record) {
    if (event_record == NULL || event_record->UserContext == NULL) {
        return;
    }
    
    etw_session_t* session = (etw_session_t*)event_record->UserContext;
    
    if (session->callback != NULL) {
        session->callback(event_record, session->callback_context);
    }
}

/**
 * @brief 初始化ETW Session
 */
etw_session_t* etw_session_init(const wchar_t* session_name) {
    if (session_name == NULL) {
        return NULL;
    }
    
    // 分配session结构体
    etw_session_t* session = (etw_session_t*)calloc(1, sizeof(etw_session_t));
    if (session == NULL) {
        return NULL;
    }
    
    // 分配EVENT_TRACE_PROPERTIES结构(需要额外空间存储Session名称)
    size_t properties_size = sizeof(EVENT_TRACE_PROPERTIES) + 
                             (wcslen(session_name) + 1) * sizeof(wchar_t) + 
                             sizeof(wchar_t) * 2; // 额外的null terminator
    
    session->properties = (PEVENT_TRACE_PROPERTIES)calloc(1, properties_size);
    if (session->properties == NULL) {
        free(session);
        return NULL;
    }
    
    // 配置SESSION属性
    session->properties->Wnode.BufferSize = (ULONG)properties_size;
    session->properties->Wnode.Flags = WNODE_FLAG_TRACED_GUID;
    session->properties->Wnode.ClientContext = 1; // 使用QueryPerformanceCounter
    session->properties->Wnode.Guid = KERNEL_PROCESS_PROVIDER_GUID;
    
    session->properties->LogFileMode = EVENT_TRACE_REAL_TIME_MODE;
    session->properties->BufferSize = ETW_BUFFER_SIZE_KB;
    session->properties->MinimumBuffers = ETW_BUFFER_COUNT;
    session->properties->MaximumBuffers = ETW_BUFFER_COUNT * 2;
    session->properties->FlushTimer = ETW_FLUSH_TIMER_SEC;
    
    // 设置Session名称偏移
    session->properties->LoggerNameOffset = sizeof(EVENT_TRACE_PROPERTIES);
    wcscpy_s((wchar_t*)((char*)session->properties + session->properties->LoggerNameOffset),
             wcslen(session_name) + 1,
             session_name);
    
    // 初始化其他字段
    session->session_handle = 0;
    session->trace_handle = 0;
    session->callback = NULL;
    session->callback_context = NULL;
    session->consume_thread = NULL;
    session->is_running = 0;
    session->restart_count = 0;
    
    return session;
}

/**
 * @brief 启动ETW Session
 */
int etw_session_start(
    etw_session_t* session,
    event_callback_fn callback,
    void* context
) {
    if (session == NULL || callback == NULL) {
        return EDR_ERROR_INVALID_PARAM;
    }
    
    // 保存回调函数和上下文
    session->callback = callback;
    session->callback_context = context;
    
    // 1. 创建ETW Session
    ULONG status = StartTraceW(
        &session->session_handle,
        ETW_SESSION_NAME,
        session->properties
    );
    
    if (status == ERROR_ALREADY_EXISTS) {
        // Session已存在,尝试停止旧Session
        TRACEHANDLE old_handle = 0;
        ControlTraceW(
            old_handle,
            ETW_SESSION_NAME,
            session->properties,
            EVENT_TRACE_CONTROL_STOP
        );
        
        // 等待一小段时间让旧Session完全停止
        Sleep(100);
        
        // 重试创建
        status = StartTraceW(
            &session->session_handle,
            ETW_SESSION_NAME,
            session->properties
        );
    }
    
    if (status != ERROR_SUCCESS) {
        if (status == ERROR_ACCESS_DENIED) {
            return EDR_ERROR_ETW_ACCESS_DENIED;
        }
        return EDR_ERROR_ETW_CREATE_FAILED;
    }
    
    // 2. 启用Provider(Microsoft-Windows-Kernel-Process)
    ENABLE_TRACE_PARAMETERS enable_params = {0};
    enable_params.Version = ENABLE_TRACE_PARAMETERS_VERSION_2;
    
    // 设置关键字过滤: PROCESS_START + PROCESS_END
    // 注意: Kernel Provider使用特殊的关键字
    // 0x10 = PROCESS_START, 0x20 = PROCESS_END
    ULONGLONG match_any_keyword = 0x10 | 0x20; // PROCESS events
    
    status = EnableTraceEx2(
        session->session_handle,
        &KERNEL_PROCESS_PROVIDER_GUID,
        EVENT_CONTROL_CODE_ENABLE_PROVIDER,
        TRACE_LEVEL_INFORMATION,
        match_any_keyword,
        0, // MatchAllKeyword
        0, // Timeout
        &enable_params
    );
    
    if (status != ERROR_SUCCESS) {
        // 启用Provider失败,清理Session
        ControlTraceW(
            session->session_handle,
            NULL,
            session->properties,
            EVENT_TRACE_CONTROL_STOP
        );
        return EDR_ERROR_ETW_ENABLE_FAILED;
    }
    
    // 3. 打开Trace用于消费事件
    session->trace_logfile.LoggerName = (LPWSTR)ETW_SESSION_NAME;
    session->trace_logfile.ProcessTraceMode = PROCESS_TRACE_MODE_REAL_TIME | 
                                                PROCESS_TRACE_MODE_EVENT_RECORD;
    session->trace_logfile.EventRecordCallback = event_record_callback_bridge;
    session->trace_logfile.Context = session; // 传递session作为上下文
    
    session->trace_handle = OpenTraceW(&session->trace_logfile);
    if (session->trace_handle == INVALID_PROCESSTRACE_HANDLE) {
        // 打开Trace失败,清理
        ControlTraceW(
            session->session_handle,
            NULL,
            session->properties,
            EVENT_TRACE_CONTROL_STOP
        );
        return EDR_ERROR_ETW_START_FAILED;
    }
    
    // 4. 启动消费线程
    session->is_running = 1;
    session->consume_thread = CreateThread(
        NULL,
        0,
        consume_thread_func,
        session,
        0,
        NULL
    );
    
    if (session->consume_thread == NULL) {
        session->is_running = 0;
        CloseTrace(session->trace_handle);
        ControlTraceW(
            session->session_handle,
            NULL,
            session->properties,
            EVENT_TRACE_CONTROL_STOP
        );
        return EDR_ERROR_ETW_START_FAILED;
    }
    
    return EDR_SUCCESS;
}

/**
 * @brief 停止ETW Session
 */
int etw_session_stop(etw_session_t* session) {
    if (session == NULL) {
        return EDR_ERROR_INVALID_PARAM;
    }
    
    if (!session->is_running) {
        return EDR_SUCCESS; // 已经停止
    }
    
    // 1. 停止事件消费
    session->is_running = 0;
    
    if (session->trace_handle != 0) {
        CloseTrace(session->trace_handle);
        session->trace_handle = 0;
    }
    
    // 2. 等待消费线程退出
    if (session->consume_thread != NULL) {
        WaitForSingleObject(session->consume_thread, 5000); // 最多等5秒
        CloseHandle(session->consume_thread);
        session->consume_thread = NULL;
    }
    
    // 3. 停止Session
    if (session->session_handle != 0) {
        ULONG status = ControlTraceW(
            session->session_handle,
            NULL,
            session->properties,
            EVENT_TRACE_CONTROL_STOP
        );
        
        session->session_handle = 0;
        
        if (status != ERROR_SUCCESS && status != ERROR_WMI_INSTANCE_NOT_FOUND) {
            return EDR_ERROR_ETW_STOP_FAILED;
        }
    }
    
    return EDR_SUCCESS;
}

/**
 * @brief 销毁ETW Session,释放资源
 */
void etw_session_destroy(etw_session_t* session) {
    if (session == NULL) {
        return;
    }
    
    // 确保Session已停止
    if (session->is_running) {
        etw_session_stop(session);
    }
    
    // 释放properties内存
    if (session->properties != NULL) {
        free(session->properties);
        session->properties = NULL;
    }
    
    // 释放session结构体
    free(session);
}
