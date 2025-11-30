# ============================================================
# EDR Core C Library - Tests CMakeLists.txt
# ============================================================
cmake_minimum_required(VERSION 3.20)

# ============================================================
# 测试源文件
# ============================================================
file(GLOB TEST_SOURCES "*.c")

# ============================================================
# 测试可执行文件
# ============================================================
if(TEST_SOURCES)
    foreach(TEST_SOURCE ${TEST_SOURCES})
        get_filename_component(TEST_NAME ${TEST_SOURCE} NAME_WE)
        add_executable(${TEST_NAME} ${TEST_SOURCE})
        target_link_libraries(${TEST_NAME} edr_core)
        target_include_directories(${TEST_NAME} PRIVATE ${CMAKE_SOURCE_DIR}/include)
        add_test(NAME ${TEST_NAME} COMMAND ${TEST_NAME})
    endforeach()
endif()

# ============================================================
# 占位测试 (确保目录非空)
# ============================================================
# TODO: 添加单元测试
# - test_collector.c
# - test_detector.c
# - test_response.c
