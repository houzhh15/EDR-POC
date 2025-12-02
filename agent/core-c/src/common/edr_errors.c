/**
 * @file edr_errors.c
 * @brief EDR错误码辅助函数实现
 * @version 1.0
 * @date 2025-12-02
 */

#include "edr_errors.h"

/**
 * @brief 获取错误消息描述
 * @param error_code 错误码
 * @return 错误描述字符串（静态字符串，不需要释放）
 */
const char* edr_error_string(int error_code) {
    switch (error_code) {
        /* 成功和通用错误 */
        case EDR_SUCCESS:
            return "Success";
        case EDR_ERROR_UNKNOWN:
            return "Unknown error";
        case EDR_ERROR_INVALID_PARAM:
            return "Invalid parameter";
        case EDR_ERROR_OUT_OF_MEMORY:
            return "Out of memory";
        case EDR_ERROR_NOT_INITIALIZED:
            return "Not initialized";
        case EDR_ERROR_TIMEOUT:
            return "Operation timeout";
        case EDR_ERROR_ACCESS_DENIED:
            return "Access denied";
        case EDR_ERROR_NOT_SUPPORTED:
            return "Operation not supported";
        
        /* ETW错误 */
        case EDR_ERROR_ETW_SESSION_EXISTS:
            return "ETW session already exists";
        case EDR_ERROR_ETW_CREATE_FAILED:
            return "ETW session creation failed";
        case EDR_ERROR_ETW_ENABLE_FAILED:
            return "ETW provider enable failed";
        case EDR_ERROR_ETW_START_FAILED:
            return "ETW session start failed";
        case EDR_ERROR_ETW_STOP_FAILED:
            return "ETW session stop failed";
        case EDR_ERROR_ETW_ACCESS_DENIED:
            return "ETW access denied (Administrator required)";
        case EDR_ERROR_ETW_NOT_RUNNING:
            return "ETW session not running";
        case EDR_ERROR_ETW_PROCESS_FAILED:
            return "ETW event processing failed";
        
        /* Event Consumer错误 */
        case EDR_ERROR_PARSE_FAILED:
            return "Event parsing failed";
        case EDR_ERROR_OPEN_PROCESS_FAILED:
            return "Failed to open process";
        case EDR_ERROR_QUERY_PROCESS_FAILED:
            return "Failed to query process information";
        case EDR_ERROR_GET_TOKEN_FAILED:
            return "Failed to get process token";
        case EDR_ERROR_HASH_FAILED:
            return "Hash calculation failed";
        case EDR_ERROR_CMDLINE_FAILED:
            return "Failed to get command line";
        case EDR_ERROR_USERNAME_FAILED:
            return "Failed to get username";
        
        /* Buffer错误 */
        case EDR_ERROR_BUFFER_FULL:
            return "Ring buffer full (event dropped)";
        case EDR_ERROR_BUFFER_EMPTY:
            return "Ring buffer empty";
        case EDR_ERROR_BUFFER_CORRUPTED:
            return "Ring buffer data corrupted";
        case EDR_ERROR_BUFFER_CREATE_FAILED:
            return "Ring buffer creation failed";
        case EDR_ERROR_BUFFER_DESTROY_FAILED:
            return "Ring buffer destroy failed";
        
        default:
            return "Unknown error code";
    }
}
