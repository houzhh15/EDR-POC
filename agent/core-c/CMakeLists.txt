# ============================================================
# EDR Core C Library - CMakeLists.txt
# ============================================================
cmake_minimum_required(VERSION 3.20)
project(edr_core C)

# ============================================================
# C 标准设置
# ============================================================
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# ============================================================
# 编译选项
# ============================================================
if(MSVC)
    add_compile_options(/W4 /WX)
else()
    add_compile_options(-Wall -Wextra -Werror -fPIC)
endif()

# ============================================================
# 平台检测
# ============================================================
if(WIN32)
    set(PLATFORM_DIR "windows")
    set(PLATFORM_LIBS "")
    message(STATUS "Target platform: Windows")
elseif(APPLE)
    set(PLATFORM_DIR "macos")
    set(PLATFORM_LIBS "-framework Security -framework CoreFoundation")
    message(STATUS "Target platform: macOS")
else()
    set(PLATFORM_DIR "linux")
    set(PLATFORM_LIBS "bpf")  # 动态链接 libbpf (LGPL 合规)
    message(STATUS "Target platform: Linux")
endif()

# ============================================================
# 源文件收集
# ============================================================
file(GLOB_RECURSE COMMON_SOURCES "src/common/*.c")
file(GLOB_RECURSE COLLECTOR_SOURCES "src/collector/${PLATFORM_DIR}/*.c")
file(GLOB_RECURSE DETECTOR_SOURCES "src/detector/*.c")
file(GLOB_RECURSE RESPONSE_SOURCES "src/response/*.c")

# PAL 平台抽象层源文件 (条件编译)
if(WIN32)
    set(PAL_SOURCES "src/pal/pal_windows.c")
elseif(APPLE)
    set(PAL_SOURCES "src/pal/pal_macos.c")
else()
    set(PAL_SOURCES "src/pal/pal_linux.c")
endif()

# 事件队列源文件
file(GLOB QUEUE_SOURCES "src/queue/*.c")

# 插件架构源文件
file(GLOB PLUGIN_SOURCES "src/plugin/*.c")

# 合并所有源文件
set(ALL_SOURCES
    ${COMMON_SOURCES}
    ${COLLECTOR_SOURCES}
    ${DETECTOR_SOURCES}
    ${RESPONSE_SOURCES}
    ${PAL_SOURCES}
    ${QUEUE_SOURCES}
    ${PLUGIN_SOURCES}
)

# 输出源文件信息
message(STATUS "Common sources: ${COMMON_SOURCES}")
message(STATUS "Collector sources (${PLATFORM_DIR}): ${COLLECTOR_SOURCES}")
message(STATUS "Detector sources: ${DETECTOR_SOURCES}")
message(STATUS "Response sources: ${RESPONSE_SOURCES}")
message(STATUS "PAL sources: ${PAL_SOURCES}")
message(STATUS "Queue sources: ${QUEUE_SOURCES}")
message(STATUS "Plugin sources: ${PLUGIN_SOURCES}")


# ============================================================
# 头文件路径
# ============================================================
include_directories(include)
include_directories(src)
include_directories(src/queue)
include_directories(src/plugin)

# ============================================================
# 共享库目标
# ============================================================
add_library(edr_core SHARED ${ALL_SOURCES})

# ============================================================
# 链接库
# ============================================================
# 查找 YARA 库
find_library(YARA_LIB yara)
if(YARA_LIB)
    target_link_libraries(edr_core ${YARA_LIB})
    message(STATUS "Found YARA library: ${YARA_LIB}")
else()
    message(WARNING "YARA library not found, detection features may be limited")
endif()

# 平台特定链接
if(PLATFORM_LIBS)
    target_link_libraries(edr_core ${PLATFORM_LIBS})
endif()

# pthread 库支持
find_package(Threads REQUIRED)
target_link_libraries(edr_core Threads::Threads)


# ============================================================
# 版本信息
# ============================================================
set_target_properties(edr_core PROPERTIES
    VERSION 0.1.0
    SOVERSION 0
)

# ============================================================
# 安装规则
# ============================================================
install(TARGETS edr_core
    LIBRARY DESTINATION lib
    RUNTIME DESTINATION bin
    ARCHIVE DESTINATION lib
)

install(DIRECTORY include/
    DESTINATION include
    FILES_MATCHING PATTERN "*.h"
)

# ============================================================
# 测试配置
# ============================================================
enable_testing()
add_subdirectory(tests)
