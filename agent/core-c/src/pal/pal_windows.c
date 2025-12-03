/**
 * @file pal_windows.c
 * @brief Platform Abstraction Layer - Windows 实现
 *
 * @copyright Copyright (c) 2024 EDR Project
 * @license Apache-2.0
 */

#include "pal.h"
#include <windows.h>
#include <tlhelp32.h>
#include <stdio.h>
#include <stdbool.h>
#include <string.h>
#include <stdint.h>

/* ============================================================
 * 全局变量
 * ============================================================ */

/* PAL 初始化标志 */
static bool g_pal_initialized = false;

/* QueryPerformanceCounter 频率 (Hz) */
static LARGE_INTEGER g_qpc_frequency = { 0 };

/* ============================================================
 * 内部辅助函数
 * ============================================================ */

/**
 * @brief 获取Windows错误消息
 * @param error_code Windows错误码
 * @param buffer 输出缓冲区
 * @param buffer_size 缓冲区大小
 * @return 成功返回true,失败返回false
 */
static bool get_windows_error_message(DWORD error_code, char* buffer, size_t buffer_size) {
    if (buffer == NULL || buffer_size == 0) {
        return false;
    }

    DWORD result = FormatMessageA(
        FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
        NULL,
        error_code,
        MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
        buffer,
        (DWORD)buffer_size,
        NULL
    );

    return result != 0;
}

/**
 * @brief 检查Windows版本 (要求Windows 10+)
 * @return 符合版本要求返回true,否则返回false
 */
static bool check_windows_version(void) {
    /* 使用 RtlGetVersion 获取真实版本 (绕过兼容性设置) */
    typedef LONG (WINAPI *RtlGetVersionPtr)(PRTL_OSVERSIONINFOW);
    
    HMODULE ntdll = GetModuleHandleA("ntdll.dll");
    if (ntdll == NULL) {
        return false;
    }

    RtlGetVersionPtr RtlGetVersion = (RtlGetVersionPtr)GetProcAddress(ntdll, "RtlGetVersion");
    if (RtlGetVersion == NULL) {
        return false;
    }

    RTL_OSVERSIONINFOW osvi;
    ZeroMemory(&osvi, sizeof(osvi));
    osvi.dwOSVersionInfoSize = sizeof(osvi);

    LONG status = RtlGetVersion(&osvi);
    if (status != 0) {
        return false;
    }

    /* Windows 10 的版本号是 10.0 */
    if (osvi.dwMajorVersion < 10) {
        fprintf(stderr, "ERROR: Windows version %lu.%lu not supported, requires Windows 10 or later\n",
                osvi.dwMajorVersion, osvi.dwMinorVersion);
        return false;
    }

    return true;
}

/* ============================================================
 * PAL 初始化/清理
 * ============================================================ */

edr_error_t pal_init(void) {
    /* 检查重复初始化 */
    if (g_pal_initialized) {
        fprintf(stderr, "ERROR: PAL already initialized\n");
        return EDR_ERR_ALREADY_INITIALIZED;
    }

    /* 检查Windows版本 */
    if (!check_windows_version()) {
        fprintf(stderr, "ERROR: Windows version check failed\n");
        return EDR_ERR_NOT_SUPPORTED;
    }

    /* 初始化高精度计数器频率 */
    if (!QueryPerformanceFrequency(&g_qpc_frequency)) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: QueryPerformanceFrequency failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: QueryPerformanceFrequency failed with code: %lu\n", error_code);
        }
        return EDR_ERR_PLATFORM;
    }

    /* 验证频率非零 */
    if (g_qpc_frequency.QuadPart == 0) {
        fprintf(stderr, "ERROR: QueryPerformanceFrequency returned zero frequency\n");
        return EDR_ERR_PLATFORM;
    }

    /* 设置初始化标志 */
    g_pal_initialized = true;

    printf("INFO: PAL initialized successfully (QPC frequency: %lld Hz)\n", g_qpc_frequency.QuadPart);
    return EDR_OK;
}

void pal_cleanup(void) {
    if (!g_pal_initialized) {
        return;
    }

    /* 重置全局状态 */
    g_pal_initialized = false;
    g_qpc_frequency.QuadPart = 0;

    printf("INFO: PAL cleanup completed\n");
}

/* ============================================================
 * 内部数据结构
 * ============================================================ */

/* 互斥锁结构 */
struct pal_mutex {
    CRITICAL_SECTION cs;
};

/* 线程结构 */
struct pal_thread {
    HANDLE handle;
    DWORD thread_id;
    bool joined;
};

/* 线程入口包装器参数 */
typedef struct {
    pal_thread_func_t user_func;
    void* user_arg;
} thread_wrapper_arg_t;

/* ============================================================
 * 互斥锁实现
 * ============================================================ */

pal_mutex_t pal_mutex_create(void) {
    struct pal_mutex* mutex = (struct pal_mutex*)malloc(sizeof(struct pal_mutex));
    if (mutex == NULL) {
        fprintf(stderr, "ERROR: Failed to allocate memory for mutex\n");
        return NULL;
    }

    InitializeCriticalSection(&mutex->cs);
    return mutex;
}

void pal_mutex_destroy(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return;
    }

    DeleteCriticalSection(&mutex->cs);
    free(mutex);
}

edr_error_t pal_mutex_lock(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    EnterCriticalSection(&mutex->cs);
    return EDR_OK;
}

edr_error_t pal_mutex_unlock(pal_mutex_t mutex) {
    if (mutex == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    LeaveCriticalSection(&mutex->cs);
    return EDR_OK;
}

/* ============================================================
 * 线程实现
 * ============================================================ */

/**
 * @brief 线程入口函数包装器
 * @param arg thread_wrapper_arg_t 指针
 * @return 线程退出码
 */
static DWORD WINAPI thread_wrapper(LPVOID arg) {
    thread_wrapper_arg_t* wrapper_arg = (thread_wrapper_arg_t*)arg;
    if (wrapper_arg == NULL) {
        return 1;
    }

    pal_thread_func_t user_func = wrapper_arg->user_func;
    void* user_arg = wrapper_arg->user_arg;
    free(wrapper_arg);

    /* 调用用户线程函数 */
    void* result = user_func(user_arg);
    
    /* Windows 线程返回 DWORD,将指针转换为整数 */
    return (DWORD)(uintptr_t)result;
}

pal_thread_t pal_thread_create(pal_thread_func_t func, void* arg) {
    if (func == NULL) {
        fprintf(stderr, "ERROR: Thread function is NULL\n");
        return NULL;
    }

    /* 分配线程结构 */
    struct pal_thread* thread = (struct pal_thread*)malloc(sizeof(struct pal_thread));
    if (thread == NULL) {
        fprintf(stderr, "ERROR: Failed to allocate memory for thread\n");
        return NULL;
    }

    /* 分配包装器参数 */
    thread_wrapper_arg_t* wrapper_arg = (thread_wrapper_arg_t*)malloc(sizeof(thread_wrapper_arg_t));
    if (wrapper_arg == NULL) {
        fprintf(stderr, "ERROR: Failed to allocate memory for thread wrapper\n");
        free(thread);
        return NULL;
    }

    wrapper_arg->user_func = func;
    wrapper_arg->user_arg = arg;

    /* 创建线程 */
    thread->handle = CreateThread(
        NULL,                   /* 默认安全属性 */
        0,                      /* 默认堆栈大小 */
        thread_wrapper,         /* 线程函数 */
        wrapper_arg,            /* 线程参数 */
        0,                      /* 立即运行 */
        &thread->thread_id      /* 返回线程ID */
    );

    if (thread->handle == NULL) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: CreateThread failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: CreateThread failed with code: %lu\n", error_code);
        }
        free(wrapper_arg);
        free(thread);
        return NULL;
    }

    thread->joined = false;
    return thread;
}

edr_error_t pal_thread_join(pal_thread_t thread, void** result) {
    if (thread == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    if (thread->joined) {
        fprintf(stderr, "ERROR: Thread already joined\n");
        return EDR_ERR_INVALID_STATE;
    }

    /* 等待线程结束 */
    DWORD wait_result = WaitForSingleObject(thread->handle, INFINITE);
    if (wait_result != WAIT_OBJECT_0) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: WaitForSingleObject failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: WaitForSingleObject failed with code: %lu\n", error_code);
        }
        return EDR_ERR_PLATFORM;
    }

    /* 获取线程退出码 */
    if (result != NULL) {
        DWORD exit_code;
        if (GetExitCodeThread(thread->handle, &exit_code)) {
            *result = (void*)(uintptr_t)exit_code;
        } else {
            *result = NULL;
        }
    }

    thread->joined = true;
    return EDR_OK;
}

void pal_thread_destroy(pal_thread_t thread) {
    if (thread == NULL) {
        return;
    }

    /* 如果未 join,先关闭句柄 */
    if (!thread->joined && thread->handle != NULL) {
        CloseHandle(thread->handle);
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

    HANDLE heap = GetProcessHeap();
    if (heap == NULL) {
        fprintf(stderr, "ERROR: GetProcessHeap failed\n");
        return NULL;
    }

    void* ptr = HeapAlloc(heap, 0, size);
    if (ptr == NULL) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: HeapAlloc failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: HeapAlloc failed with code: %lu\n", error_code);
        }
    }

    return ptr;
}

void pal_mem_free(void* ptr) {
    if (ptr == NULL) {
        return;
    }

    HANDLE heap = GetProcessHeap();
    if (heap == NULL) {
        fprintf(stderr, "ERROR: GetProcessHeap failed in free\n");
        return;
    }

    if (!HeapFree(heap, 0, ptr)) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: HeapFree failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: HeapFree failed with code: %lu\n", error_code);
        }
    }
}

void* pal_mem_calloc(size_t count, size_t size) {
    if (count == 0 || size == 0) {
        return NULL;
    }

    /* 检查乘法溢出 */
    if (size > SIZE_MAX / count) {
        fprintf(stderr, "ERROR: calloc size overflow\n");
        return NULL;
    }

    HANDLE heap = GetProcessHeap();
    if (heap == NULL) {
        fprintf(stderr, "ERROR: GetProcessHeap failed\n");
        return NULL;
    }

    /* HEAP_ZERO_MEMORY 标志自动清零内存 */
    void* ptr = HeapAlloc(heap, HEAP_ZERO_MEMORY, count * size);
    if (ptr == NULL) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: HeapAlloc (calloc) failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: HeapAlloc (calloc) failed with code: %lu\n", error_code);
        }
    }

    return ptr;
}

/* ============================================================
 * 时间实现
 * ============================================================ */

uint64_t pal_time_now_ms(void) {
    if (!g_pal_initialized) {
        fprintf(stderr, "ERROR: PAL not initialized\n");
        return 0;
    }

    LARGE_INTEGER counter;
    if (!QueryPerformanceCounter(&counter)) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: QueryPerformanceCounter failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: QueryPerformanceCounter failed with code: %lu\n", error_code);
        }
        return 0;
    }

    /* 转换为毫秒: (counter * 1000) / frequency */
    uint64_t ms = (counter.QuadPart * 1000) / g_qpc_frequency.QuadPart;
    return ms;
}

void pal_sleep_ms(uint32_t ms) {
    Sleep(ms);
}

/* ============================================================
 * 文件操作实现
 * ============================================================ */

edr_error_t pal_file_read(const char* path, void* buf, size_t size, size_t* bytes_read) {
    if (path == NULL || buf == NULL || bytes_read == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    if (size == 0) {
        *bytes_read = 0;
        return EDR_OK;
    }

    /* 打开文件 */
    HANDLE file = CreateFileA(
        path,
        GENERIC_READ,
        FILE_SHARE_READ,
        NULL,
        OPEN_EXISTING,
        FILE_ATTRIBUTE_NORMAL,
        NULL
    );

    if (file == INVALID_HANDLE_VALUE) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: CreateFileA failed for '%s': %s (code: %lu)\n",
                    path, error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: CreateFileA failed for '%s' with code: %lu\n",
                    path, error_code);
        }
        return EDR_ERR_IO;
    }

    /* 读取文件 */
    DWORD read_bytes = 0;
    BOOL success = ReadFile(file, buf, (DWORD)size, &read_bytes, NULL);
    DWORD error_code = GetLastError();

    CloseHandle(file);

    if (!success) {
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: ReadFile failed for '%s': %s (code: %lu)\n",
                    path, error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: ReadFile failed for '%s' with code: %lu\n",
                    path, error_code);
        }
        return EDR_ERR_IO;
    }

    *bytes_read = read_bytes;
    return EDR_OK;
}

edr_error_t pal_file_move(const char* src, const char* dst) {
    if (src == NULL || dst == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    /* MOVEFILE_REPLACE_EXISTING 允许覆盖已存在文件 */
    if (!MoveFileExA(src, dst, MOVEFILE_REPLACE_EXISTING)) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: MoveFileExA failed from '%s' to '%s': %s (code: %lu)\n",
                    src, dst, error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: MoveFileExA failed from '%s' to '%s' with code: %lu\n",
                    src, dst, error_code);
        }
        return EDR_ERR_IO;
    }

    return EDR_OK;
}

bool pal_file_exists(const char* path) {
    if (path == NULL) {
        return false;
    }

    DWORD attributes = GetFileAttributesA(path);
    return (attributes != INVALID_FILE_ATTRIBUTES);
}

/* ============================================================
 * 进程管理实现
 * ============================================================ */

edr_error_t pal_process_get_list(pal_process_info_t* list, size_t max_count, size_t* actual_count) {
    if (list == NULL || actual_count == NULL) {
        return EDR_ERR_INVALID_PARAM;
    }

    *actual_count = 0;

    if (max_count == 0) {
        return EDR_OK;
    }

    /* 创建进程快照 */
    HANDLE snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if (snapshot == INVALID_HANDLE_VALUE) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: CreateToolhelp32Snapshot failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: CreateToolhelp32Snapshot failed with code: %lu\n", error_code);
        }
        return EDR_ERR_PLATFORM;
    }

    PROCESSENTRY32 pe32;
    pe32.dwSize = sizeof(PROCESSENTRY32);

    /* 获取第一个进程 */
    if (!Process32First(snapshot, &pe32)) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: Process32First failed: %s (code: %lu)\n",
                    error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: Process32First failed with code: %lu\n", error_code);
        }
        CloseHandle(snapshot);
        return EDR_ERR_PLATFORM;
    }

    /* 遍历进程列表 */
    size_t count = 0;
    do {
        if (count < max_count) {
            list[count].pid = pe32.th32ProcessID;
            list[count].ppid = pe32.th32ParentProcessID;

            /* 复制进程名称，确保不溢出 */
            size_t name_len = strlen(pe32.szExeFile);
            if (name_len >= sizeof(list[count].name)) {
                name_len = sizeof(list[count].name) - 1;
            }
            memcpy(list[count].name, pe32.szExeFile, name_len);
            list[count].name[name_len] = '\0';
        }
        count++;
    } while (Process32Next(snapshot, &pe32));

    CloseHandle(snapshot);

    *actual_count = count;
    return EDR_OK;
}

edr_error_t pal_process_terminate(uint32_t pid) {
    if (pid == 0) {
        return EDR_ERR_INVALID_PARAM;
    }

    /* 打开进程句柄 */
    HANDLE process = OpenProcess(PROCESS_TERMINATE, FALSE, pid);
    if (process == NULL) {
        DWORD error_code = GetLastError();
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: OpenProcess failed for PID %u: %s (code: %lu)\n",
                    pid, error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: OpenProcess failed for PID %u with code: %lu\n",
                    pid, error_code);
        }
        return EDR_ERR_PLATFORM;
    }

    /* 终止进程 */
    BOOL success = TerminateProcess(process, 1);
    DWORD error_code = GetLastError();

    CloseHandle(process);

    if (!success) {
        char error_msg[256];
        if (get_windows_error_message(error_code, error_msg, sizeof(error_msg))) {
            fprintf(stderr, "ERROR: TerminateProcess failed for PID %u: %s (code: %lu)\n",
                    pid, error_msg, error_code);
        } else {
            fprintf(stderr, "ERROR: TerminateProcess failed for PID %u with code: %lu\n",
                    pid, error_code);
        }
        return EDR_ERR_PLATFORM;
    }

    return EDR_OK;
}
