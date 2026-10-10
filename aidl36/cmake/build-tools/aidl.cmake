#
# Copyright © 2022 Github Lzhiyong
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

if(NOT DEFINED CDH_BUILD_TOOLS_ROOT OR NOT IS_DIRECTORY "${CDH_BUILD_TOOLS_ROOT}")
    message(FATAL_ERROR "CDH_BUILD_TOOLS_ROOT must point to the checked-out AOSP build tools")
endif()
if(NOT DEFINED CDH_BUILD_TOOLS_HOST OR NOT CDH_BUILD_TOOLS_HOST MATCHES "^linux-(arm64|x86)$")
    message(FATAL_ERROR "CDH_BUILD_TOOLS_HOST must select the native AOSP Linux tools")
endif()
foreach(CDH_VERSION IN ITEMS CDH_BISON_VERSION CDH_FLEX_VERSION CDH_M4_VERSION)
    if(NOT DEFINED ${CDH_VERSION} OR NOT "${${CDH_VERSION}}" MATCHES "^[0-9]+\\.[0-9]+\\.[0-9]+$")
        message(FATAL_ERROR "${CDH_VERSION} must select an exact generator version")
    endif()
endforeach()
set(CDH_TOOLS_BIN ${CDH_BUILD_TOOLS_ROOT}/${CDH_BUILD_TOOLS_HOST}/bin)
set(BISON_EXECUTABLE ${CDH_TOOLS_BIN}/bison)
set(FLEX_EXECUTABLE ${CDH_TOOLS_BIN}/flex)
set(CDH_M4_BIN ${CDH_TOOLS_BIN}/m4)
set(CDH_BISON_DATA ${CDH_BUILD_TOOLS_ROOT}/common/bison)
foreach(CDH_GENERATOR IN ITEMS ${BISON_EXECUTABLE} ${FLEX_EXECUTABLE} ${CDH_M4_BIN})
    if(NOT EXISTS "${CDH_GENERATOR}")
        message(FATAL_ERROR "AOSP generator is unavailable: ${CDH_GENERATOR}")
    endif()
endforeach()
find_package(BISON ${CDH_BISON_VERSION} EXACT REQUIRED)
find_package(FLEX ${CDH_FLEX_VERSION} EXACT REQUIRED)
execute_process(
    COMMAND "${CDH_M4_BIN}" --version
    OUTPUT_VARIABLE CDH_M4_OUTPUT
    OUTPUT_STRIP_TRAILING_WHITESPACE
    COMMAND_ERROR_IS_FATAL ANY
    )
string(REGEX MATCH "^[^\n\r]*" CDH_M4_LINE "${CDH_M4_OUTPUT}")
if(NOT CDH_M4_LINE STREQUAL "m4 (GNU M4) ${CDH_M4_VERSION}")
    message(FATAL_ERROR "AOSP M4 ${CDH_M4_VERSION} is required")
endif()
if(NOT IS_DIRECTORY "${CDH_BISON_DATA}")
    message(FATAL_ERROR "AOSP Bison package data directory is unavailable")
endif()

set(CDH_PARSER_CPP ${CMAKE_CURRENT_BINARY_DIR}/aidl_language_y.cpp)
set(CDH_PARSER_H ${CMAKE_CURRENT_BINARY_DIR}/aidl_language_y.h)
set(CDH_LOCATION_H ${CMAKE_CURRENT_BINARY_DIR}/location.hh)
set(CDH_POSITION_H ${CMAKE_CURRENT_BINARY_DIR}/position.hh)
add_custom_command(
    OUTPUT "${CDH_PARSER_CPP}" "${CDH_PARSER_H}" "${CDH_LOCATION_H}" "${CDH_POSITION_H}"
    COMMAND "${CMAKE_COMMAND}" -E env
        "BISON_PKGDATADIR=${CDH_BISON_DATA}"
        "M4=${CDH_M4_BIN}"
        "${BISON_EXECUTABLE}"
        -d
        --defines=${CDH_PARSER_H}
        -o ${CDH_PARSER_CPP}
        ${SRC}/aidl/aidl_language_y.yy
    DEPENDS "${SRC}/aidl/aidl_language_y.yy"
    VERBATIM
    )

set(CDH_SCANNER_CPP ${CMAKE_CURRENT_BINARY_DIR}/aidl_language_l.cpp)
set(CDH_SCANNER_H ${CMAKE_CURRENT_BINARY_DIR}/aidl_language_l.h)
add_custom_command(
    OUTPUT "${CDH_SCANNER_CPP}" "${CDH_SCANNER_H}"
    COMMAND "${CMAKE_COMMAND}" -E env
        "M4=${CDH_M4_BIN}"
        "${FLEX_EXECUTABLE}"
        --header-file=${CDH_SCANNER_H}
        -o ${CDH_SCANNER_CPP}
        ${SRC}/aidl/aidl_language_l.ll
    DEPENDS "${SRC}/aidl/aidl_language_l.ll" "${CDH_PARSER_H}"
    VERBATIM
    )
add_executable(aidl
    ${SRC}/aidl/aidl_checkapi.cpp
    ${SRC}/aidl/aidl_const_expressions.cpp
    ${SRC}/aidl/aidl_dumpapi.cpp
    ${SRC}/aidl/aidl_language.cpp
    ${SRC}/aidl/aidl_to_common.cpp
    ${SRC}/aidl/aidl_to_cpp_common.cpp
    ${SRC}/aidl/aidl_to_cpp.cpp
    ${SRC}/aidl/aidl_to_java.cpp
    ${SRC}/aidl/aidl_to_ndk.cpp
    ${SRC}/aidl/aidl_to_rust.cpp
    ${SRC}/aidl/aidl_typenames.cpp
    ${SRC}/aidl/aidl.cpp
    ${SRC}/aidl/ast_java.cpp
    ${SRC}/aidl/check_valid.cpp
    ${SRC}/aidl/code_writer.cpp
    ${SRC}/aidl/comments.cpp
    ${SRC}/aidl/diagnostics.cpp
    ${SRC}/aidl/generate_aidl_mappings.cpp
    ${SRC}/aidl/generate_cpp.cpp
    ${SRC}/aidl/generate_cpp_analyzer.cpp
    ${SRC}/aidl/generate_java_binder.cpp
    ${SRC}/aidl/generate_java.cpp
    ${SRC}/aidl/generate_ndk.cpp
    ${SRC}/aidl/generate_rust.cpp
    ${SRC}/aidl/import_resolver.cpp
    ${SRC}/aidl/io_delegate.cpp
    ${SRC}/aidl/location.cpp
    ${SRC}/aidl/logging.cpp
    ${SRC}/aidl/options.cpp
    ${SRC}/aidl/parser.cpp
    ${SRC}/aidl/permission.cpp
    ${SRC}/aidl/preprocess.cpp
    ${SRC}/aidl/main.cpp
    ${CDH_PARSER_CPP}
    ${CDH_PARSER_H}
    ${CDH_LOCATION_H}
    ${CDH_POSITION_H}
    ${CDH_SCANNER_CPP}
    ${CDH_SCANNER_H}
    )

target_include_directories(aidl PRIVATE
    ${CMAKE_CURRENT_BINARY_DIR}
    ${SRC}/aidl
    ${SRC}/aidl/include
    ${SRC}/libbase/include
    ${SRC}/fmtlib/include
    ${SRC}/googletest/googletest/include
    )

target_link_libraries(aidl
    libbase
    liblog
    gtest
    fmt::fmt
    )
