/**
 * @file pal_macos.c
 * @brief Platform Abstraction Layer - macOS 实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "pal.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <mach/mach_time.h>
#include <signal.h>
#include <libproc.h>
#include <sys/sysctl.h>

/* ============================================================
 * 内部结构定义
 * ============================================================ */

struct pal_mutex {
    pthread_mutex_t mutex;
};

struct pal_thread {
    pthread_t thread;
    bool joined;
};

/* 时间转换因子 (用于 mach_absolute_time) */
static mach_timebase_info_data_t g_timebase_info;
static bool g_pal_initialized = false;

/* ============================================================
 * PAL 初始化/清理
 * ============================================================ */

edr_error_t pal_init(void) {
    if (g_pal_initialized) {
        return EDR_ERR_ALREADY_INITIALIZED;
    }
    
    /* 获取时间基准信息 */
    mach_timebase_info(&g_timebase_info);
    
    g_pal_initialized = true;
    return EDR_OK;
}

void pal_cleanup(void) {
    g_pal_initialized = false;
}

/* ============================================================
 * 互斥锁实现
 * ============================================================ */

pal_mutex_t pal_mutex_create(void) {
    struct pal_mutex* mutex = malloc(sizeof(struct pal_mutex));
    if (mutex == NULL) {
        return NULL;
    }
    
    if (pthread_mutex_init(&mutex->mutex, NULL) != 0) {
        free(mutex);
        return NULL;
    }
    
    return mutex;
}

void pal_mutex_destroy(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return;
    }
    
    pthread_mutex_destroy(&mutex->mutex);
    free(mutex);
}

edr_error_t pal_mutex_lock(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (pthread_mutex_lock(&mutex->mutex) != 0) {
        return EDR_ERR_UNKNOWN;
    }
    
    return EDR_OK;
}

edr_error_t pal_mutex_unlock(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (pthread_mutex_unlock(&mutex->mutex) != 0) {
        return EDR_ERR_UNKNOWN;
    }
    
    return EDR_OK;
}

/* ============================================================
 * 线程实现
 * ============================================================ */

pal_thread_t pal_thread_create(pal_thread_func_t func, void* arg) {
    if (func == NULL) {
        return NULL;
    }
    
    struct pal_thread* thread = malloc(sizeof(struct pal_thread));
    if (thread == NULL) {
        return NULL;
    }
    
    thread->joined = false;
    
    if (pthread_create(&thread->thread, NULL, func, arg) != 0) {
        free(thread);
        return NULL;
    }
    
    return thread;
}

edr_error_t pal_thread_join(pal_thread_t thread, void** result) {
    if (thread == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (thread->joined) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (pthread_join(thread->thread, result) != 0) {
        return EDR_ERR_UNKNOWN;
    }
    
    thread->joined = true;
    return EDR_OK;
}

void pal_thread_destroy(pal_thread_t thread) {
    if (thread == NULL) {
        return;
    }
    
    free(thread);
}

/* ============================================================
 * 内存管理实现
 * ============================================================ */

void* pal_mem_alloc(size_t size) {
    if (size == 0) {
        return NULL;
    }
    return malloc(size);
}

void pal_mem_free(void* ptr) {
    free(ptr);
}

void* pal_mem_calloc(size_t count, size_t size) {
    if (count == 0 || size == 0) {
        return NULL;
    }
    return calloc(count, size);
}

/* ============================================================
 * 时间实现
 * ============================================================ */

uint64_t pal_time_now_ms(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (uint64_t)tv.tv_sec * 1000 + (uint64_t)tv.tv_usec / 1000;
}

void pal_sleep_ms(uint32_t ms) {
    usleep((useconds_t)ms * 1000);
}

/* ============================================================
 * 文件操作实现
 * ============================================================ */

edr_error_t pal_file_read(const char* path, void* buf, size_t size, size_t* bytes_read) {
    if (path == NULL || buf == NULL || size == 0) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    FILE* fp = fopen(path, "rb");
    if (fp == NULL) {
        return EDR_ERR_PERMISSION;
    }
    
    size_t read_count = fread(buf, 1, size, fp);
    fclose(fp);
    
    if (bytes_read != NULL) {
        *bytes_read = read_count;
    }
    
    return EDR_OK;
}

edr_error_t pal_file_move(const char* src, const char* dst) {
    if (src == NULL || dst == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (rename(src, dst) != 0) {
        return EDR_ERR_PERMISSION;
    }
    
    return EDR_OK;
}

bool pal_file_exists(const char* path) {
    if (path == NULL) {
        return false;
    }
    
    struct stat st;
    return stat(path, &st) == 0;
}

/* ============================================================
 * 进程管理实现
 * ============================================================ */

edr_error_t pal_process_get_list(pal_process_info_t* list, size_t max_count, size_t* actual_count) {
    if (list == NULL || max_count == 0) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    /* 获取所有进程 PID */
    int pid_count = proc_listallpids(NULL, 0);
    if (pid_count <= 0) {
        if (actual_count != NULL) {
            *actual_count = 0;
        }
        return EDR_OK;
    }
    
    pid_t* pids = malloc(sizeof(pid_t) * pid_count);
    if (pids == NULL) {
        return EDR_ERR_NO_MEMORY;
    }
    
    pid_count = proc_listallpids(pids, sizeof(pid_t) * pid_count);
    
    size_t count = 0;
    for (int i = 0; i < pid_count && count < max_count; i++) {
        struct proc_bsdinfo info;
        int ret = proc_pidinfo(pids[i], PROC_PIDTBSDINFO, 0, &info, sizeof(info));
        if (ret <= 0) {
            continue;
        }
        
        list[count].pid = pids[i];
        list[count].ppid = info.pbi_ppid;
        strncpy(list[count].name, info.pbi_comm, sizeof(list[count].name) - 1);
        list[count].name[sizeof(list[count].name) - 1] = '\0';
        
        /* 获取可执行文件路径 */
        char pathbuf[PROC_PIDPATHINFO_MAXSIZE];
        if (proc_pidpath(pids[i], pathbuf, sizeof(pathbuf)) > 0) {
            strncpy(list[count].path, pathbuf, sizeof(list[count].path) - 1);
            list[count].path[sizeof(list[count].path) - 1] = '\0';
        } else {
            list[count].path[0] = '\0';
        }
        
        count++;
    }
    
    free(pids);
    
    if (actual_count != NULL) {
        *actual_count = count;
    }
    
    return EDR_OK;
}

edr_error_t pal_process_terminate(uint32_t pid) {
    if (pid == 0) {
        return EDR_ERR_INVALID_PARAM;
    }
    
    if (kill((pid_t)pid, SIGTERM) != 0) {
        return EDR_ERR_PERMISSION;
    }
    
    return EDR_OK;
}
