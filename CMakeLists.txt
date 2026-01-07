# LL-journal CMake Build Configuration
# Go journal service

if(NOT GO_EXECUTABLE)
    message(FATAL_ERROR "Go executable not found. Required for LL-journal build.")
endif()

# Build directory
set(LL_JOURNAL_SOURCE_DIR ${CMAKE_CURRENT_SOURCE_DIR})
set(LL_JOURNAL_BINARY_DIR ${CMAKE_CURRENT_BINARY_DIR})

# Build flags
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(GO_BUILD_FLAGS "-ldflags=-s -w")  # Strip symbols and debug info
else()
    set(GO_BUILD_FLAGS "")
endif()

# Output binary path
set(LL_JOURNAL_BINARY_PATH ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/ll-journal)

# Custom command to build LL-journal
# Disable CGO to avoid issues with missing kernel headers and for static builds
add_custom_command(
    OUTPUT ${LL_JOURNAL_BINARY_PATH}
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 ${GO_EXECUTABLE} build
        ${GO_BUILD_FLAGS}
        -o ${LL_JOURNAL_BINARY_PATH}
        ./cmd/ll-journal
    WORKING_DIRECTORY ${LL_JOURNAL_SOURCE_DIR}
    COMMENT "Building LL-journal (Go)"
    DEPENDS
        ${LL_JOURNAL_SOURCE_DIR}/go.mod
        ${LL_JOURNAL_SOURCE_DIR}/go.sum
)

# Custom target
add_custom_target(ll-journal
    DEPENDS ${LL_JOURNAL_BINARY_PATH}
    COMMENT "LL-journal build target"
)

# Test target
if(BUILD_TESTS)
    add_custom_target(ll-journal-test
        COMMAND ${GO_EXECUTABLE} test ./...
        WORKING_DIRECTORY ${LL_JOURNAL_SOURCE_DIR}
        COMMENT "Running LL-journal tests"
    )
endif()

# Coverage target
if(BUILD_COVERAGE)
    add_custom_target(ll-journal-coverage
        COMMAND ${GO_EXECUTABLE} test -coverprofile=${CMAKE_BINARY_DIR}/coverage/ll-journal-coverage.out ./...
        COMMAND ${GO_EXECUTABLE} tool cover -html=${CMAKE_BINARY_DIR}/coverage/ll-journal-coverage.out
            -o ${CMAKE_BINARY_DIR}/coverage/ll-journal-coverage.html
        WORKING_DIRECTORY ${LL_JOURNAL_SOURCE_DIR}
        COMMENT "Generating LL-journal code coverage"
    )
endif()

# Clean target
add_custom_target(ll-journal-clean
    COMMAND ${CMAKE_COMMAND} -E remove ${LL_JOURNAL_BINARY_PATH}
    COMMENT "Cleaning LL-journal build artifacts"
)

# Install target
install(
    FILES ${LL_JOURNAL_BINARY_PATH}
    DESTINATION bin
    RENAME ll-journal
    OPTIONAL
)

# Print build info
message(STATUS "LL-journal Configuration:")
message(STATUS "  Source: ${LL_JOURNAL_SOURCE_DIR}")
message(STATUS "  Binary: ${LL_JOURNAL_BINARY_PATH}")
