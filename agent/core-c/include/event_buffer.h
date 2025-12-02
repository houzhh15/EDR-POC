/**
 * @file event_buffer.h
 * @brief 事件缓冲队列 - SPSC无锁环形缓冲区
 * 
 * 实现单生产者单消费者(Single Producer Single Consumer)模式的无锁环形缓冲区,
 * 用于C层ETW Consumer和Go层CGO之间的高效事件传递。
 * 
 * 特性:
 * - 固定容量4096个事件(约24MB内存)
 * - 使用原子操作保证线程安全,无需mutex
 * - 非阻塞push/pop,buffer满时丢弃新事件
 * - 支持批量pop,减少CGO调用开销
 * - 提供统计信息(总数、丢弃数、使用率)
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#ifndef EDR_EVENT_BUFFER_H
#define EDR_EVENT_BUFFER_H

#ifdef __cplusplus
extern "C" {
#endif

#include "edr_events.h"
#include <stdint.h>

/**
 * @brief 环形缓冲区容量(必须是2的幂次,便于取模优化)
 */
#define EVENT_BUFFER_SIZE 4096

/**
 * @brief 事件缓冲队列结构体
 * 
 * 内存布局:
 * - events数组: 4096 × 6KB ≈ 24MB (最坏情况)
 * - 索引和统计: 48 bytes
 * - 总计: ~24MB
 */
typedef struct {
    /** 事件存储数组 */
    edr_process_event_t events[EVENT_BUFFER_SIZE];
    
    /** 写入位置索引(生产者修改,消费者只读) */
    volatile uint32_t write_pos;
    
    /** 读取位置索引(消费者修改,生产者只读) */
    volatile uint32_t read_pos;
    
    /** 总推送事件数(用于统计,原子递增) */
    volatile uint64_t total_pushed;
    
    /** 总弹出事件数(用于统计,原子递增) */
    volatile uint64_t total_popped;
    
    /** 丢弃事件计数(buffer满时递增) */
    volatile uint64_t dropped_count;
    
    /** 峰值使用量(0-4096) */
    volatile uint32_t peak_usage;
    
} event_buffer_t;

/**
 * @brief 创建事件缓冲区
 * 
 * 分配并初始化event_buffer_t结构体,将所有索引和统计清零。
 * 
 * @return 成功返回buffer指针,失败返回NULL
 * @note 调用者负责通过event_buffer_destroy()释放内存
 */
event_buffer_t* event_buffer_create(void);

/**
 * @brief 销毁事件缓冲区
 * 
 * 释放buffer占用的内存。
 * 
 * @param buffer 要销毁的buffer指针
 * @note 销毁后指针失效,不可再使用
 */
void event_buffer_destroy(event_buffer_t* buffer);

/**
 * @brief 推送单个事件到缓冲区(非阻塞)
 * 
 * 生产者接口。将事件写入write_pos位置,然后原子地更新write_pos。
 * 如果buffer已满,则丢弃事件并增加dropped_count计数。
 * 
 * 算法:
 * 1. 计算下一个写位置: next_write = (write_pos + 1) % SIZE
 * 2. 检查是否满: next_write == read_pos
 * 3. 如果满,增加dropped_count并返回错误
 * 4. 否则,memcpy事件到events[write_pos]
 * 5. 原子更新write_pos = next_write
 * 6. 原子递增total_pushed
 * 
 * @param buffer 缓冲区指针
 * @param event 要推送的事件指针(会被复制)
 * @return 0=成功, -1=buffer满(事件已丢弃)
 * @note 本函数是线程安全的(仅单个生产者线程调用)
 */
int event_buffer_push(event_buffer_t* buffer, const edr_process_event_t* event);

/**
 * @brief 弹出单个事件(非阻塞)
 * 
 * 消费者接口。从read_pos位置读取事件,然后原子地更新read_pos。
 * 如果buffer为空,返回0(无数据)。
 * 
 * 算法:
 * 1. 检查是否空: read_pos == write_pos
 * 2. 如果空,返回0
 * 3. 否则,memcpy events[read_pos]到out_event
 * 4. 计算下一个读位置: next_read = (read_pos + 1) % SIZE
 * 5. 原子更新read_pos = next_read
 * 6. 原子递增total_popped
 * 
 * @param buffer 缓冲区指针
 * @param out_event 输出事件指针(调用者分配内存)
 * @return 1=成功弹出, 0=buffer为空
 * @note 本函数是线程安全的(仅单个消费者线程调用)
 */
int event_buffer_pop(event_buffer_t* buffer, edr_process_event_t* out_event);

/**
 * @brief 批量弹出事件(非阻塞)
 * 
 * 消费者接口。一次性弹出最多max_count个事件,实际弹出数量取决于buffer中的可用事件。
 * 用于减少CGO调用次数,提升性能。
 * 
 * @param buffer 缓冲区指针
 * @param events 输出事件数组指针(调用者分配内存,至少max_count个元素)
 * @param max_count 最多弹出事件数(建议100)
 * @return 实际弹出的事件数(0-max_count)
 * @note 本函数是线程安全的(仅单个消费者线程调用)
 */
int event_buffer_pop_batch(
    event_buffer_t* buffer,
    edr_process_event_t* events,
    int max_count
);

/**
 * @brief 获取缓冲区统计信息
 * 
 * 查询buffer的运行统计,用于监控和性能分析。
 * 
 * @param buffer 缓冲区指针
 * @param out_total_pushed 输出总推送数(可为NULL)
 * @param out_total_popped 输出总弹出数(可为NULL)
 * @param out_dropped 输出丢弃数(可为NULL)
 * @param out_usage_percent 输出当前使用率百分比(0-100,可为NULL)
 * @note 所有统计值都是原子读取,保证一致性
 */
void event_buffer_get_stats(
    event_buffer_t* buffer,
    uint64_t* out_total_pushed,
    uint64_t* out_total_popped,
    uint64_t* out_dropped,
    uint32_t* out_usage_percent
);

/**
 * @brief 获取当前缓冲区使用量
 * 
 * 计算公式: (write_pos - read_pos + SIZE) % SIZE
 * 
 * @param buffer 缓冲区指针
 * @return 当前使用的事件数量(0-4095)
 */
static inline uint32_t event_buffer_get_usage(const event_buffer_t* buffer) {
    uint32_t w = buffer->write_pos;
    uint32_t r = buffer->read_pos;
    return (w >= r) ? (w - r) : (EVENT_BUFFER_SIZE - r + w);
}

/**
 * @brief 检查缓冲区是否为空
 * 
 * @param buffer 缓冲区指针
 * @return 1=空, 0=非空
 */
static inline int event_buffer_is_empty(const event_buffer_t* buffer) {
    return buffer->read_pos == buffer->write_pos;
}

/**
 * @brief 检查缓冲区是否已满
 * 
 * @param buffer 缓冲区指针
 * @return 1=满, 0=未满
 */
static inline int event_buffer_is_full(const event_buffer_t* buffer) {
    uint32_t next_write = (buffer->write_pos + 1) % EVENT_BUFFER_SIZE;
    return next_write == buffer->read_pos;
}

#ifdef __cplusplus
}
#endif

#endif /* EDR_EVENT_BUFFER_H */
