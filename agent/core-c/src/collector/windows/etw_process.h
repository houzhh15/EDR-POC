/**
 * @file etw_process.h
 * @brief 进程事件消费模块 - ETW进程事件解析和处理
 * 
 * 接收ETW Session的进程事件回调,解析EVENT_RECORD结构,提取进程元数据,
 * 查询进程句柄获取完整信息(路径、命令行、用户、哈希),并推送到Ring Buffer。
 * 
 * 核心功能:
 * - 解析PROCESS_START事件(PID/PPID/Name/Path/CmdLine/User/Hash)
 * - 解析PROCESS_END事件(PID/ExitCode)
 * - 进程句柄LRU缓存(256个)避免重复查询
 * - 哈希计算超时控制(10ms)
 * - 快速非阻塞回调,立即返回给ETW
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifndef EDR_ETW_PROCESS_H
#define EDR_ETW_PROCESS_H

#ifdef __cplusplus
extern "C" {
#endif

#include "etw_session.h"
#include "../../include/event_buffer.h"
#include "../../include/edr_events.h"
#include <windows.h>
#include <stdint.h>

/**
 * @brief 进程句柄缓存大小
 */
#define PROCESS_HANDLE_CACHE_SIZE 256

/**
 * @brief 哈希计算超时时间(毫秒)
 */
#define HASH_CALC_TIMEOUT_MS 10

/**
 * @brief 进程句柄缓存项
 */
typedef struct {
    uint32_t pid;           // 进程ID
    HANDLE handle;          // 进程句柄
    uint64_t last_access;   // 最后访问时间(用于LRU)
} process_handle_cache_entry_t;

/**
 * @brief 进程事件消费者结构体
 */
typedef struct {
    /** 关联的ETW Session */
    etw_session_t* session;
    
    /** 事件缓冲区 */
    event_buffer_t* buffer;
    
    /** 进程句柄LRU缓存 */
    process_handle_cache_entry_t handle_cache[PROCESS_HANDLE_CACHE_SIZE];
    
    /** 缓存中已使用的槽位数 */
    int cache_used;
    
    /** 总事件数统计 */
    volatile uint64_t total_events;
    
    /** 解析失败统计 */
    volatile uint64_t parse_errors;
    
} etw_process_consumer_t;

/**
 * @brief 创建进程事件消费者
 * 
 * @param buffer 事件缓冲区指针
 * @return 成功返回consumer指针,失败返回NULL
 * @note 调用者负责销毁
 */
etw_process_consumer_t* etw_process_consumer_create(event_buffer_t* buffer);

/**
 * @brief 销毁进程事件消费者
 * 
 * 释放资源,关闭所有缓存的进程句柄。
 * 
 * @param consumer 消费者指针
 */
void etw_process_consumer_destroy(etw_process_consumer_t* consumer);

/**
 * @brief ETW事件回调函数(由ETW Session调用)
 * 
 * 这是核心回调函数,在ETW消费线程中被调用。
 * 根据OpCode判断事件类型(1=START, 2=END),调用对应的解析函数。
 * 解析成功后调用event_buffer_push()写入ring buffer。
 * 
 * 注意: 此函数必须快速返回,不能阻塞ETW事件流。
 * 
 * @param event_record ETW事件记录
 * @param context 用户上下文(etw_process_consumer_t指针)
 */
void etw_process_event_callback(PEVENT_RECORD event_record, void* context);

/**
 * @brief 解析PROCESS_START事件
 * 
 * 提取进程创建事件的所有字段:
 * - 从EVENT_HEADER提取: PID, PPID, 创建时间
 * - 从EVENT_DATA提取: ImageFileName(进程名称)
 * - 通过OpenProcess获取进程句柄
 * - 通过QueryFullProcessImageName获取完整路径
 * - 通过ReadProcessMemory读取PEB获取命令行
 * - 通过OpenProcessToken+GetTokenInformation获取用户信息
 * - 计算SHA256哈希(超时10ms)
 * 
 * @param consumer 消费者对象
 * @param event_record ETW事件记录
 * @param out_event 输出事件结构体
 * @return 0=成功, 非0=错误码
 */
int etw_parse_process_start(
    etw_process_consumer_t* consumer,
    PEVENT_RECORD event_record,
    edr_process_event_t* out_event
);

/**
 * @brief 解析PROCESS_END事件
 * 
 * 提取进程终止事件的基本字段:
 * - 从EVENT_HEADER提取: PID, 结束时间
 * - 从EVENT_DATA提取: ExitCode
 * 
 * 注意: 进程已结束无法查询句柄,仅填充基本信息。
 * 
 * @param consumer 消费者对象
 * @param event_record ETW事件记录
 * @param out_event 输出事件结构体
 * @return 0=成功, 非0=错误码
 */
int etw_parse_process_end(
    etw_process_consumer_t* consumer,
    PEVENT_RECORD event_record,
    edr_process_event_t* out_event
);

/**
 * @brief 从缓存获取或打开进程句柄
 * 
 * 使用LRU策略缓存进程句柄,避免重复OpenProcess调用。
 * 
 * @param consumer 消费者对象
 * @param pid 进程ID
 * @return 成功返回进程句柄,失败返回NULL
 */
HANDLE etw_get_process_handle(etw_process_consumer_t* consumer, uint32_t pid);

#ifdef __cplusplus
}
#endif

#endif /* EDR_ETW_PROCESS_H */
