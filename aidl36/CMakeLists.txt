# Copyright (c) 2025-2026 fei_cong(https://github.com/feicong/feicong-course)
# Portions adapted from android-sdk-tools, Copyright 2022 Github Lzhiyong.
# Licensed under the Apache License, Version 2.0.

# Build the Android 16 AIDL compiler for the native Linux host.
# The per-library layout is adapted from lzhiyong/android-sdk-tools.

cmake_minimum_required(VERSION 3.20)
project(cdh-aidl-host LANGUAGES C CXX ASM)

set(CMAKE_C_STANDARD 11)
set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(CheckCXXCompilerFlag)
check_cxx_compiler_flag(-fcolor-diagnostics HAVE_COLOR_DIAG)
if(HAVE_COLOR_DIAG)
    add_compile_options(-fcolor-diagnostics)
endif()
add_compile_options(-fPIC -Wno-attributes)
add_compile_options(-Wall -Wextra -Werror -Wno-vla-cxx-extension -Wno-c99-designator -Wno-missing-designated-field-initializers -Wno-character-conversion)
if(NOT DEFINED CDH_PLATFORM_SDK OR NOT CDH_PLATFORM_SDK MATCHES "^[1-9][0-9]*$")
    message(FATAL_ERROR "CDH_PLATFORM_SDK must be a positive integer")
endif()
add_compile_definitions(PLATFORM_SDK_VERSION=${CDH_PLATFORM_SDK})
add_compile_definitions(_FILE_OFFSET_BITS=64)
add_compile_definitions(_GNU_SOURCE)

# The caller supplies the checked-out Android 16 source set.
if(NOT DEFINED AIDL_SOURCE_ROOT OR NOT IS_DIRECTORY "${AIDL_SOURCE_ROOT}")
    message(FATAL_ERROR "AIDL_SOURCE_ROOT must point to the checked-out Android 16 source set")
endif()
set(SRC "${AIDL_SOURCE_ROOT}")

set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
set(FMT_OS OFF CACHE BOOL "" FORCE)

add_subdirectory(${SRC}/fmtlib ${CMAKE_BINARY_DIR}/deps/fmtlib EXCLUDE_FROM_ALL)
add_subdirectory(${SRC}/googletest ${CMAKE_BINARY_DIR}/deps/googletest EXCLUDE_FROM_ALL)

include(cmake/lib/liblog.cmake)
include(cmake/lib/libbase.cmake)
include(cmake/build-tools/aidl.cmake)
