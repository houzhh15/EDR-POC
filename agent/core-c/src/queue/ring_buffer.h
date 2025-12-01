/**
 * @file ring_buffer.h
 * @brief SPSC 无锁环形缓冲队列
 *
 * 单生产者单消费者 (Single-Producer Single-Consumer) 无锁队列实现。
 * 用于 C 采集器线程到 CGO 回调的高效事件传递。
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#ifndef EDR_RING_BUFFER_H
#define EDR_RING_BUFFER_H

#ifdef __cplusplus
extern "C" {
#endif

#include "edr_core.h"
#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdatomic.h>

/* ============================================================
 * 事件结构定义
 * ============================================================ */

/**
 * @brief 事件结构 (柔性数组)
 *
 * 内存布局: [type(4) | timestamp(8) | data_len(4) | data(data_len)]
 */
typedef struct edr_event {
    uint32_t type;          /**< 事件类型 */
    uint64_t timestamp;     /**< 时间戳 (毫秒) */
    uint32_t data_len;      /**< 数据长度 */
    char data[];            /**< 柔性数组: JSON 数据 */
} edr_event_t;

/**
 * @brief 创建事件
 * @param type 事件类型
 * @param timestamp 时间戳
 * @param data 数据指针
 * @param data_len 数据长度
 * @return 事件指针，失败返回 NULL (调用者负责释放)
 */
edr_event_t* edr_event_create(uint32_t type, uint64_t timestamp, const char* data, uint32_t data_len);

/**
 * @brief 释放事件
 * @param event 事件指针
 */
void edr_event_destroy(edr_event_t* event);

/* ============================================================
 * 环形缓冲队列结构定义
 * ============================================================ */

/**
 * @brief SPSC 环形缓冲队列
 */
typedef struct edr_ring_buffer {
    edr_event_t** slots;        /**< 事件指针数组 */
    uint32_t capacity;          /**< 队列容量 (必须是 2 的幂) */
    uint32_t mask;              /**< capacity - 1, 用于位运算取模 */
    _Atomic uint32_t head;      /**< 写索引 (生产者) */
    _Atomic uint32_t tail;      /**< 读索引 (消费者) */
} edr_ring_buffer_t;

/* ============================================================
 * 队列操作接口
 * ============================================================ */

/**
 * @brief 创建环形缓冲队列
 * @param capacity 队列容量 (必须是 2 的幂)
 * @return 队列指针，失败返回 NULL
 */
edr_ring_buffer_t* ring_buffer_create(uint32_t capacity);

/**
 * @brief 销毁环形缓冲队列
 * @param rb 队列指针
 *
 * 会释放所有未消费的事件
 */
void ring_buffer_destroy(edr_ring_buffer_t* rb);

/**
 * @brief 入队 (生产者调用)
 * @param rb 队列指针
 * @param event 事件指针 (队列获取所有权)
 * @return EDR_OK 成功，EDR_ERR_NO_MEMORY 队列满
 */
edr_error_t ring_buffer_push(edr_ring_buffer_t* rb, edr_event_t* event);

/**
 * @brief 出队 (消费者调用)
 * @param rb 队列指针
 * @return 事件指针，队列空返回 NULL (调用者获取所有权)
 */
edr_event_t* ring_buffer_pop(edr_ring_buffer_t* rb);

/**
 * @brief 检查队列是否满
 * @param rb 队列指针
 * @return true 满，false 未满
 */
bool ring_buffer_is_full(edr_ring_buffer_t* rb);

/**
 * @brief 检查队列是否空
 * @param rb 队列指针
 * @return true 空，false 非空
 */
bool ring_buffer_is_empty(edr_ring_buffer_t* rb);

/**
 * @brief 获取当前队列大小
 * @param rb 队列指针
 * @return 队列中的事件数量
 */
uint32_t ring_buffer_size(edr_ring_buffer_t* rb);

#ifdef __cplusplus
}
#endif

#endif /* EDR_RING_BUFFER_H */
