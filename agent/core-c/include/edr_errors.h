/**
 * @file edr_errors.h
 * @brief EDR错误码定义
 * @version 1.0
 * @date 2025-12-02
 * 
 * 定义EDR系统的错误码体系，用于错误处理和日志记录
 * 错误码分区:
 * - 0: 成功
 * - -1 ~ -99: 通用错误
 * - -100 ~ -199: ETW相关错误
 * - -200 ~ -299: Event Consumer错误
 * - -300 ~ -399: Buffer错误
 */

#ifndef EDR_ERRORS_H
#define EDR_ERRORS_H

#ifdef __cplusplus
extern "C" {
#endif

/* ========== 成功和通用错误 (0 ~ -99) ========== */

/** @brief 操作成功 */
#define EDR_SUCCESS                 0

/** @brief 未知错误 */
#define EDR_ERROR_UNKNOWN          -1

/** @brief 无效参数 */
#define EDR_ERROR_INVALID_PARAM    -2

/** @brief 内存不足 */
#define EDR_ERROR_OUT_OF_MEMORY    -3

/** @brief 未初始化 */
#define EDR_ERROR_NOT_INITIALIZED  -4

/** @brief 操作超时 */
#define EDR_ERROR_TIMEOUT          -5

/** @brief 权限不足 */
#define EDR_ERROR_ACCESS_DENIED    -6

/** @brief 操作不支持 */
#define EDR_ERROR_NOT_SUPPORTED    -7

/* ========== ETW Session错误 (-100 ~ -199) ========== */

/** @brief ETW Session已存在 */
#define EDR_ERROR_ETW_SESSION_EXISTS     -100

/** @brief ETW Session创建失败 */
#define EDR_ERROR_ETW_CREATE_FAILED      -101

/** @brief ETW Provider启用失败 */
#define EDR_ERROR_ETW_ENABLE_FAILED      -102

/** @brief ETW Session启动失败 */
#define EDR_ERROR_ETW_START_FAILED       -103

/** @brief ETW Session停止失败 */
#define EDR_ERROR_ETW_STOP_FAILED        -104

/** @brief ETW访问权限不足（需要管理员权限） */
#define EDR_ERROR_ETW_ACCESS_DENIED      -105

/** @brief ETW Session未运行 */
#define EDR_ERROR_ETW_NOT_RUNNING        -106

/** @brief ETW事件处理失败 */
#define EDR_ERROR_ETW_PROCESS_FAILED     -107

/* ========== Event Consumer错误 (-200 ~ -299) ========== */

/** @brief 事件解析失败 */
#define EDR_ERROR_PARSE_FAILED           -200

/** @brief 打开进程失败 */
#define EDR_ERROR_OPEN_PROCESS_FAILED    -201

/** @brief 查询进程信息失败 */
#define EDR_ERROR_QUERY_PROCESS_FAILED   -202

/** @brief 获取进程Token失败 */
#define EDR_ERROR_GET_TOKEN_FAILED       -203

/** @brief 哈希计算失败 */
#define EDR_ERROR_HASH_FAILED            -204

/** @brief 命令行获取失败 */
#define EDR_ERROR_CMDLINE_FAILED         -205

/** @brief 用户名获取失败 */
#define EDR_ERROR_USERNAME_FAILED        -206

/* ========== Ring Buffer错误 (-300 ~ -399) ========== */

/** @brief Buffer已满（事件被丢弃） */
#define EDR_ERROR_BUFFER_FULL            -300

/** @brief Buffer为空（无事件可读） */
#define EDR_ERROR_BUFFER_EMPTY           -301

/** @brief Buffer数据损坏 */
#define EDR_ERROR_BUFFER_CORRUPTED       -302

/** @brief Buffer创建失败 */
#define EDR_ERROR_BUFFER_CREATE_FAILED   -303

/** @brief Buffer销毁失败 */
#define EDR_ERROR_BUFFER_DESTROY_FAILED  -304

/**
 * @brief 获取错误消息描述
 * @param error_code 错误码
 * @return 错误描述字符串（静态字符串，不需要释放）
 */
const char* edr_error_string(int error_code);

#ifdef __cplusplus
}
#endif

#endif /* EDR_ERRORS_H */
