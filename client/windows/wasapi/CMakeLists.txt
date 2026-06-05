cmake_minimum_required(VERSION 3.14)
project(wasapi_capture LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)

add_library(wasapi_capture SHARED
    wasapi_capture.cpp
)

target_link_libraries(wasapi_capture
    ole32
)

# Copy DLL to the same output directory as the Flutter executable
set_target_properties(wasapi_capture PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/runner/$<CONFIG>"
)
