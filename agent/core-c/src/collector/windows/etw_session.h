/**
 * @file etw_session.h
 * @brief ETW Session管理模块 - Windows进程事件跟踪会话
 * 
 * 管理ETW(Event Tracing for Windows)实时跟踪会话的完整生命周期,
 * 连接到Microsoft-Windows-Kernel-Process Provider,采集进程创建和终止事件。
 * 
 * 核心功能:
 * - 创建和管理ETW实时跟踪会话
 * - 连接到Kernel-Process Provider
 * - 配置事件过滤规则(PROCESS_START/END)
 * - 在独立线程中消费ETW事件
 * - 自动错误恢复(Session重启,最多3次)
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifndef EDR_ETW_SESSION_H
#define EDR_ETW_SESSION_H

#ifdef __cplusplus
extern "C" {
#endif

#include <windows.h>
#include <evntrace.h>
#include <stdint.h>

/**
 * @brief ETW Session配置常量
 */
#define ETW_SESSION_NAME        L"EDR-Process-Collector-Session"
#define ETW_BUFFER_SIZE_KB      64
#define ETW_BUFFER_COUNT        20
#define ETW_FLUSH_TIMER_SEC     1
#define ETW_MAX_RESTART_RETRY   3

/**
 * @brief Kernel Process Provider GUID
 * {22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716}
 */
static const GUID KERNEL_PROCESS_PROVIDER_GUID = 
    {0x22fb2cd6, 0x0e7b, 0x422b, {0xa0, 0xc7, 0x2f, 0xad, 0x1f, 0xd0, 0xe7, 0x16}};

/**
 * @brief 事件回调函数类型
 * 
 * @param event_record ETW事件记录指针
 * @param context 用户上下文指针(通常是consumer对象)
 */
typedef void (*event_callback_fn)(PEVENT_RECORD event_record, void* context);

/**
 * @brief ETW Session结构体
 * 
 * 封装ETW跟踪会话的所有状态和句柄。
 */
typedef struct {
    /** ETW Session句柄(StartTrace返回) */
    TRACEHANDLE session_handle;
    
    /** Trace消费句柄(OpenTrace返回) */
    TRACEHANDLE trace_handle;
    
    /** Session属性结构(EVENT_TRACE_PROPERTIES) */
    PEVENT_TRACE_PROPERTIES properties;
    
    /** Logfile配置结构 */
    EVENT_TRACE_LOGFILEW trace_logfile;
    
    /** 事件回调函数指针 */
    event_callback_fn callback;
    
    /** 回调函数上下文(传递给callback) */
    void* callback_context;
    
    /** 事件消费线程句柄 */
    HANDLE consume_thread;
    
    /** 运行状态标志(0=stopped, 1=running) */
    volatile int is_running;
    
    /** Session重启计数器 */
    int restart_count;
    
} etw_session_t;

/**
 * @brief 初始化ETW Session
 * 
 * 分配并初始化etw_session_t结构体,配置Session属性。
 * 不实际创建Session,仅准备数据结构。
 * 
 * @param session_name Session名称(通常使用ETW_SESSION_NAME)
 * @return 成功返回session指针,失败返回NULL
 * @note 调用者负责通过etw_session_destroy()释放内存
 */
etw_session_t* etw_session_init(const wchar_t* session_name);

/**
 * @brief 启动ETW Session(非阻塞)
 * 
 * 执行以下操作:
 * 1. 使用StartTrace()创建实时Session
 * 2. 如果Session已存在,先关闭旧Session再重试
 * 3. 使用EnableTraceEx2()连接到Kernel-Process Provider
 * 4. 设置关键字过滤(PROCESS_START/END事件)
 * 5. 在独立线程中调用ProcessTrace()消费事件
 * 
 * @param session Session对象指针
 * @param callback 事件回调函数
 * @param context 传递给回调的用户上下文
 * @return 0=成功, 非0=错误码(参考edr_errors.h)
 * @note 函数立即返回,事件在后台线程中异步处理
 */
int etw_session_start(
    etw_session_t* session,
    event_callback_fn callback,
    void* context
);

/**
 * @brief 停止ETW Session
 * 
 * 执行以下操作:
 * 1. 调用CloseTrace()停止事件消费
 * 2. 等待消费线程退出
 * 3. 调用ControlTrace(STOP)关闭Session
 * 
 * @param session Session对象指针
 * @return 0=成功, 非0=错误码
 * @note 函数会阻塞直到Session完全停止
 */
int etw_session_stop(etw_session_t* session);

/**
 * @brief 销毁ETW Session,释放资源
 * 
 * 释放SESSION_TRACE_PROPERTIES内存和session结构体。
 * 调用前必须先调用etw_session_stop()停止Session。
 * 
 * @param session Session对象指针
 */
void etw_session_destroy(etw_session_t* session);

#ifdef __cplusplus
}
#endif

#endif /* EDR_ETW_SESSION_H */
