#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace ai_vm {

/// Represents a simple execution context for a VM instruction stream.
struct ExecutionContext {
    std::uint64_t gas_remaining = 0;
    std::uint64_t pc = 0;
    std::vector<std::uint8_t> memory;
    std::vector<std::string> trace;
};

/// A lightweight VM interpreter stub that can execute a basic opcode stream.
class VM {
public:
    /// Execute a bytecode buffer using the supplied execution context.
    static bool Execute(const std::vector<std::uint8_t>& bytecode, ExecutionContext& context);
};

}  // namespace ai_vm
