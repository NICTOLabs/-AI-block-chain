#pragma once

#include <cstdint>
#include <functional>
#include <string>
#include <vector>

namespace ai_vm {

enum class AIOpcode : std::uint8_t;

std::string OpcodeName(AIOpcode opcode);
std::uint64_t GasCostForOpcode(AIOpcode opcode);

struct ExecutionContext {
    std::uint64_t gas_remaining = 0;
    std::uint64_t pc = 0;
    std::vector<std::uint8_t> memory;
    std::vector<std::string> trace;
    std::function<bool(AIOpcode, ExecutionContext&)> handler = nullptr;
};

class VM {
public:
    static bool Execute(const std::vector<std::uint8_t>& bytecode, ExecutionContext& context);
};

}  // namespace ai_vm
