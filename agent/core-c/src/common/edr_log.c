/**
 * @file edr_log.c
 * @brief EDR日志框架实现
 * 
 * 实现日志的初始化、输出、关闭等功能。
 * 支持日志级别过滤、文件输出、格式化输出。
 * 
 * @version 1.0
 * @date 2025-12-02
 */

#include "edr_log.h"
#include <stdarg.h>
#include <string.h>
#include <time.h>

#ifdef _WIN32
#include <windows.h>
#else
#include <sys/time.h>
#endif

/**
 * @brief 全局日志配置
 */
static edr_log_config_t g_log_config = {
    .min_level = EDR_LOG_LEVEL_INFO,
    .target = EDR_LOG_TARGET_STDOUT,
    .log_file_path = {0},
    .log_file_handle = NULL
};

/**
 * @brief 日志级别字符串
 */
static const char* log_level_strings[] = {
    "DEBUG",
    "INFO ",
    "WARN ",
    "ERROR"
};

/**
 * @brief 获取当前时间戳字符串
 * 
 * @param buffer 输出缓冲区
 * @param buffer_size 缓冲区大小
 */
static void get_timestamp(char* buffer, size_t buffer_size) {
#ifdef _WIN32
    SYSTEMTIME st;
    GetLocalTime(&st);
    snprintf(buffer, buffer_size, "%04d-%02d-%02d %02d:%02d:%02d.%03d",
             st.wYear, st.wMonth, st.wDay,
             st.wHour, st.wMinute, st.wSecond, st.wMilliseconds);
#else
    struct timeval tv;
    gettimeofday(&tv, NULL);
    struct tm tm_info;
    localtime_r(&tv.tv_sec, &tm_info);
    snprintf(buffer, buffer_size, "%04d-%02d-%02d %02d:%02d:%02d.%03d",
             tm_info.tm_year + 1900, tm_info.tm_mon + 1, tm_info.tm_mday,
             tm_info.tm_hour, tm_info.tm_min, tm_info.tm_sec, (int)(tv.tv_usec / 1000));
#endif
}

/**
 * @brief 提取文件名(去除路径)
 * 
 * @param file_path 完整路径
 * @return 文件名指针
 */
static const char* extract_filename(const char* file_path) {
    const char* last_slash = strrchr(file_path, '/');
    const char* last_backslash = strrchr(file_path, '\\');
    
    const char* filename = file_path;
    if (last_slash != NULL) {
        filename = last_slash + 1;
    }
    if (last_backslash != NULL && last_backslash > filename) {
        filename = last_backslash + 1;
    }
    
    return filename;
}

/**
 * @brief 初始化日志系统
 */
int edr_log_init(edr_log_level_t min_level, edr_log_target_t target, const char* log_file_path) {
    g_log_config.min_level = min_level;
    g_log_config.target = target;
    
    if (target == EDR_LOG_TARGET_FILE) {
        if (log_file_path == NULL) {
            return -1;
        }
        
#ifdef _WIN32
        strncpy_s(g_log_config.log_file_path, sizeof(g_log_config.log_file_path),
                  log_file_path, _TRUNCATE);
        
        // 打开日志文件(追加模式)
        errno_t err = fopen_s(&g_log_config.log_file_handle, log_file_path, "a");
        if (err != 0 || g_log_config.log_file_handle == NULL) {
            return -1;
        }
#else
        strncpy(g_log_config.log_file_path, log_file_path, sizeof(g_log_config.log_file_path) - 1);
        g_log_config.log_file_path[sizeof(g_log_config.log_file_path) - 1] = '\0';
        
        // 打开日志文件(追加模式)
        g_log_config.log_file_handle = fopen(log_file_path, "a");
        if (g_log_config.log_file_handle == NULL) {
            return -1;
        }
#endif
        
        // 设置行缓冲
        setvbuf(g_log_config.log_file_handle, NULL, _IOLBF, 0);
    }
    
    return 0;
}

/**
 * @brief 关闭日志系统
 */
void edr_log_shutdown(void) {
    if (g_log_config.log_file_handle != NULL) {
        fclose(g_log_config.log_file_handle);
        g_log_config.log_file_handle = NULL;
    }
}

/**
 * @brief 核心日志函数
 */
void edr_log(edr_log_level_t level, const char* file, int line, const char* format, ...) {
    // 级别过滤
    if (level < g_log_config.min_level) {
        return;
    }
    
    // 获取时间戳 (格式: YYYY-MM-DD HH:MM:SS.mmm = 23字符 + null)
    char timestamp[64];
    get_timestamp(timestamp, sizeof(timestamp));
    
    // 提取文件名
    const char* filename = extract_filename(file);
    
    // 获取级别字符串
    const char* level_str = (level >= 0 && level <= EDR_LOG_LEVEL_ERROR) 
                            ? log_level_strings[level] 
                            : "UNKNOWN";
    
    // 选择输出目标
    FILE* output = (g_log_config.target == EDR_LOG_TARGET_FILE && g_log_config.log_file_handle != NULL)
                   ? g_log_config.log_file_handle
                   : stdout;
    
    // 输出日志头部: [时间戳] [级别] [文件:行号]
    fprintf(output, "[%s] [%s] [%s:%d] ", timestamp, level_str, filename, line);
    
    // 输出消息内容
    va_list args;
    va_start(args, format);
    vfprintf(output, format, args);
    va_end(args);
    
    // 换行
    fprintf(output, "\n");
    
    // 立即刷新(确保日志不丢失)
    fflush(output);
}
