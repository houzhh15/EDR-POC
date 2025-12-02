/**
 * @file event_buffer.c
 * @brief 事件缓冲队列实现 - SPSC无锁环形缓冲区
 * 
 * 实现细节:
 * - 使用Windows InterlockedXxx系列函数实现原子操作
 * - write_pos和read_pos使用volatile确保内存可见性
 * - 生产者(ETW Consumer)只修改write_pos,消费者(CGO)只修改read_pos
 * - 无锁算法避免了mutex开销,适合高频事件场景
 * 
 * 性能特性:
 * - Push/Pop时间复杂度: O(1)
 * - 内存占用: 固定~24MB
 * - 无动态内存分配
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#include "event_buffer.h"
#include "edr_errors.h"
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <windows.h>
#else
// 非Windows平台使用C11 stdatomic(暂不支持)
#error "Currently only Windows platform is supported"
#endif

/**
 * @brief 创建事件缓冲区
 */
event_buffer_t* event_buffer_create(void) {
    // 分配内存(使用calloc自动清零)
    event_buffer_t* buffer = (event_buffer_t*)calloc(1, sizeof(event_buffer_t));
    if (buffer == NULL) {
        return NULL;
    }
    
    // 初始化索引和统计(calloc已清零,此处仅为明确)
    buffer->write_pos = 0;
    buffer->read_pos = 0;
    buffer->total_pushed = 0;
    buffer->total_popped = 0;
    buffer->dropped_count = 0;
    buffer->peak_usage = 0;
    
    return buffer;
}

/**
 * @brief 销毁事件缓冲区
 */
void event_buffer_destroy(event_buffer_t* buffer) {
    if (buffer != NULL) {
        free(buffer);
    }
}

/**
 * @brief 推送单个事件到缓冲区(非阻塞)
 */
int event_buffer_push(event_buffer_t* buffer, const edr_process_event_t* event) {
    if (buffer == NULL || event == NULL) {
        return EDR_ERROR_INVALID_PARAM;
    }
    
    // 读取当前写位置
    uint32_t current_write = buffer->write_pos;
    uint32_t next_write = (current_write + 1) % EVENT_BUFFER_SIZE;
    
    // 检查buffer是否已满
    // 满的条件: (write_pos + 1) % SIZE == read_pos
    if (next_write == buffer->read_pos) {
        // Buffer满,丢弃事件
        InterlockedIncrement64((volatile LONG64*)&buffer->dropped_count);
        return EDR_ERROR_BUFFER_FULL;
    }
    
    // 写入事件到当前写位置
    memcpy(&buffer->events[current_write], event, sizeof(edr_process_event_t));
    
    // 原子更新写位置(确保事件完全写入后再更新索引)
    InterlockedExchange((volatile LONG*)&buffer->write_pos, next_write);
    
    // 原子递增总推送计数
    InterlockedIncrement64((volatile LONG64*)&buffer->total_pushed);
    
    // 更新峰值使用量
    uint32_t usage = event_buffer_get_usage(buffer);
    uint32_t current_peak = buffer->peak_usage;
    while (usage > current_peak) {
        // 尝试原子更新peak_usage
        LONG old_peak = InterlockedCompareExchange(
            (volatile LONG*)&buffer->peak_usage,
            usage,
            current_peak
        );
        if (old_peak == (LONG)current_peak) {
            break; // 更新成功
        }
        current_peak = buffer->peak_usage; // 重新读取
    }
    
    return EDR_SUCCESS;
}

/**
 * @brief 弹出单个事件(非阻塞)
 */
int event_buffer_pop(event_buffer_t* buffer, edr_process_event_t* out_event) {
    if (buffer == NULL || out_event == NULL) {
        return EDR_ERROR_INVALID_PARAM;
    }
    
    // 读取当前读位置
    uint32_t current_read = buffer->read_pos;
    
    // 检查buffer是否为空
    // 空的条件: read_pos == write_pos
    if (current_read == buffer->write_pos) {
        return EDR_ERROR_BUFFER_EMPTY; // Buffer空,无数据
    }
    
    // 读取事件到输出
    memcpy(out_event, &buffer->events[current_read], sizeof(edr_process_event_t));
    
    // 计算下一个读位置
    uint32_t next_read = (current_read + 1) % EVENT_BUFFER_SIZE;
    
    // 原子更新读位置
    InterlockedExchange((volatile LONG*)&buffer->read_pos, next_read);
    
    // 原子递增总弹出计数
    InterlockedIncrement64((volatile LONG64*)&buffer->total_popped);
    
    return 1; // 返回1表示成功弹出1个事件
}

/**
 * @brief 批量弹出事件(非阻塞)
 */
int event_buffer_pop_batch(
    event_buffer_t* buffer,
    edr_process_event_t* events,
    int max_count
) {
    if (buffer == NULL || events == NULL || max_count <= 0) {
        return 0;
    }
    
    int count = 0;
    
    // 循环弹出事件,直到达到max_count或buffer为空
    for (int i = 0; i < max_count; i++) {
        uint32_t current_read = buffer->read_pos;
        
        // 检查是否为空
        if (current_read == buffer->write_pos) {
            break; // Buffer空,停止弹出
        }
        
        // 读取事件
        memcpy(&events[i], &buffer->events[current_read], sizeof(edr_process_event_t));
        
        // 更新读位置
        uint32_t next_read = (current_read + 1) % EVENT_BUFFER_SIZE;
        InterlockedExchange((volatile LONG*)&buffer->read_pos, next_read);
        
        // 递增计数
        count++;
    }
    
    // 原子递增总弹出计数
    if (count > 0) {
        InterlockedExchangeAdd64((volatile LONG64*)&buffer->total_popped, count);
    }
    
    return count;
}

/**
 * @brief 获取缓冲区统计信息
 */
void event_buffer_get_stats(
    event_buffer_t* buffer,
    uint64_t* out_total_pushed,
    uint64_t* out_total_popped,
    uint64_t* out_dropped,
    uint32_t* out_usage_percent
) {
    if (buffer == NULL) {
        return;
    }
    
    // 原子读取统计值(使用InterlockedXxx确保读取一致性)
    if (out_total_pushed != NULL) {
        *out_total_pushed = InterlockedCompareExchange64(
            (volatile LONG64*)&buffer->total_pushed, 0, 0
        );
    }
    
    if (out_total_popped != NULL) {
        *out_total_popped = InterlockedCompareExchange64(
            (volatile LONG64*)&buffer->total_popped, 0, 0
        );
    }
    
    if (out_dropped != NULL) {
        *out_dropped = InterlockedCompareExchange64(
            (volatile LONG64*)&buffer->dropped_count, 0, 0
        );
    }
    
    if (out_usage_percent != NULL) {
        uint32_t usage = event_buffer_get_usage(buffer);
        *out_usage_percent = (usage * 100) / EVENT_BUFFER_SIZE;
    }
}
