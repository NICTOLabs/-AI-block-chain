cmake_minimum_required(VERSION 3.20)
project(ai_vm LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_library(ai_vm STATIC
    src/vm.cpp
    src/opcodes.cpp
    src/gas.cpp
)

target_include_directories(ai_vm PUBLIC src)

enable_testing()
add_executable(vm_tests
    src/vm_tests.cpp
    src/vm.cpp
    src/opcodes.cpp
    src/gas.cpp
)

target_include_directories(vm_tests PRIVATE src)

target_compile_definitions(vm_tests PRIVATE VM_TESTS)
