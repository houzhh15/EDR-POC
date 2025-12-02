# CMakeLists.txt for EDR Collector Module
# 配置ETW进程事件采集模块的构建

cmake_minimum_required(VERSION 3.15)

# 编译event_buffer.c(跨平台)
add_library(edr_event_buffer STATIC
    event_buffer.c
)

target_include_directories(edr_event_buffer PUBLIC
    ${CMAKE_SOURCE_DIR}/include
)

# Windows特定的ETW采集模块
if(WIN32)
    # 编译Windows ETW模块
    add_library(edr_collector_windows STATIC
        windows/etw_session.c
        windows/etw_process.c
    )
    
    target_include_directories(edr_collector_windows PUBLIC
        ${CMAKE_SOURCE_DIR}/include
        ${CMAKE_CURRENT_SOURCE_DIR}
    )
    
    # 编译定义
    target_compile_definitions(edr_collector_windows PRIVATE
        UNICODE
        _UNICODE
        WIN32_LEAN_AND_MEAN
    )
    
    # 编译选项(MSVC)
    if(MSVC)
        target_compile_options(edr_collector_windows PRIVATE
            /W4           # 警告级别4
            /WX           # 警告视为错误
            /GS           # 缓冲区安全检查
            /sdl          # SDL检查
        )
    endif()
    
    # 链接Windows库
    target_link_libraries(edr_collector_windows PUBLIC
        advapi32      # ETW API
        tdh           # Trace Data Helper
        psapi         # Process API
    )
    
    # 链接event_buffer
    target_link_libraries(edr_collector_windows PUBLIC
        edr_event_buffer
    )
endif()

# 编译公共日志模块
add_library(edr_log STATIC
    ../common/edr_log.c
    ../common/edr_errors.c
)

target_include_directories(edr_log PUBLIC
    ${CMAKE_SOURCE_DIR}/include
)

# 组合成edr_core总库(供Go CGO调用)
if(WIN32)
    add_library(edr_core STATIC
        $<TARGET_OBJECTS:edr_event_buffer>
        $<TARGET_OBJECTS:edr_collector_windows>
        $<TARGET_OBJECTS:edr_log>
    )
    
    # 设置输出目录
    set_target_properties(edr_core PROPERTIES
        ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}
    )
    
    # 链接依赖
    target_link_libraries(edr_core PUBLIC
        advapi32
        tdh
        psapi
    )
endif()
