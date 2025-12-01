/**
 * @file ring_buffer.c
 * @brief SPSC 无锁环形缓冲队列实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "ring_buffer.h"
#include <stdlib.h>
#include <string.h>

/* ============================================================
 * 辅助函数
 * ============================================================ */

/**
 * @brief 检查数值是否是 2 的幂
 */
static inline bool is_power_of_two(uint32_t n) {
    return n > 0 && (n & (n - 1)) == 0;
}

/* ============================================================
 * 事件操作实现
 * ============================================================ */

edr_event_t* edr_event_create(uint32_t type, uint64_t timestamp, const char* data, uint32_t data_len) {
    if (data == NULL && data_len > 0) {
        return NULL;
    }
    
    /* 分配事件结构 + 柔性数组空间 */
    size_t total_size = sizeof(edr_event_t) + data_len + 1;  /* +1 for null terminator */
    edr_event_t* event = malloc(total_size);
    if (event == NULL) {
        return NULL;
    }
    
    event->type = type;
    event->timestamp = timestamp;
    event->data_len = data_len;
    
    if (data != NULL && data_len > 0) {
        memcpy(event->data, data, data_len);
    }
    event->data[data_len] = '\0';  /* Null terminate */
    
    return event;
}

void edr_event_destroy(edr_event_t* event) {
    free(event);
}

/* ============================================================
 * 队列操作实现
 * ============================================================ */

edr_ring_buffer_t* ring_buffer_create(uint32_t capacity) {
    /* 容量必须是 2 的幂 */
    if (!is_power_of_two(capacity)) {
        return NULL;
    }
    
    edr_ring_buffer_t* rb = malloc(sizeof(edr_ring_buffer_t));
    if (rb == NULL) {
        return NULL;
    }
    
    rb->slots = calloc(capacity, sizeof(edr_event_t*));
    if (rb->slots == NULL) {
        free(rb);
        return NULL;
    }
    
    rb->capacity = capacity;
    rb->mask = capacity - 1;
    atomic_store(&rb->head, 0);
    atomic_store(&rb->tail, 0);
    
    return rb;
}

void ring_buffer_destroy(edr_ring_buffer_t* rb) {
    if (rb == NULL) {
        return;
    }
    
    /* 释放所有未消费的事件 */
    uint32_t tail = atomic_load(&rb->tail);
    uint32_t head = atomic_load(&rb->head);
    
    while (tail != head) {
        uint32_t index = tail & rb->mask;
        if (rb->slots[index] != NULL) {
            edr_event_destroy(rb->slots[index]);
            rb->slots[index] = NULL;
        }
        tail++;
    }
    
    free(rb->slots);
    free(rb);
}

edr_error_t ring_buffer_push(edr_ring_buffer_t* rb, edr_event_t* event) {
    if (rb == NULL || event == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    uint32_t head = atomic_load_explicit(&rb->head, memory_order_relaxed);
    uint32_t tail = atomic_load_explicit(&rb->tail, memory_order_acquire);
    
    /* 检查队列是否满 */
    if (head - tail >= rb->capacity) {
        return EDR_ERR_NO_MEMORY;
    }
    
    /* 存入事件 */
    uint32_t index = head & rb->mask;
    rb->slots[index] = event;
    
    /* 更新 head 索引 */
    atomic_store_explicit(&rb->head, head + 1, memory_order_release);
    
    return EDR_OK;
}

edr_event_t* ring_buffer_pop(edr_ring_buffer_t* rb) {
    if (rb == NULL) {
        return NULL;
    }
    
    uint32_t tail = atomic_load_explicit(&rb->tail, memory_order_relaxed);
    uint32_t head = atomic_load_explicit(&rb->head, memory_order_acquire);
    
    /* 检查队列是否空 */
    if (tail == head) {
        return NULL;
    }
    
    /* 取出事件 */
    uint32_t index = tail & rb->mask;
    edr_event_t* event = rb->slots[index];
    rb->slots[index] = NULL;
    
    /* 更新 tail 索引 */
    atomic_store_explicit(&rb->tail, tail + 1, memory_order_release);
    
    return event;
}

bool ring_buffer_is_full(edr_ring_buffer_t* rb) {
    if (rb == NULL) {
        return true;
    }
    
    uint32_t head = atomic_load_explicit(&rb->head, memory_order_relaxed);
    uint32_t tail = atomic_load_explicit(&rb->tail, memory_order_acquire);
    
    return (head - tail) >= rb->capacity;
}

bool ring_buffer_is_empty(edr_ring_buffer_t* rb) {
    if (rb == NULL) {
        return true;
    }
    
    uint32_t head = atomic_load_explicit(&rb->head, memory_order_acquire);
    uint32_t tail = atomic_load_explicit(&rb->tail, memory_order_relaxed);
    
    return tail == head;
}

uint32_t ring_buffer_size(edr_ring_buffer_t* rb) {
    if (rb == NULL) {
        return 0;
    }
    
    uint32_t head = atomic_load_explicit(&rb->head, memory_order_acquire);
    uint32_t tail = atomic_load_explicit(&rb->tail, memory_order_relaxed);
    
    return head - tail;
}
