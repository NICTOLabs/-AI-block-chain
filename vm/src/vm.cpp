#include "vm.h"

#include <algorithm>
#include <stdexcept>

#include "opcodes.h"

namespace ai_vm {

namespace {

bool IsAIInstruction(std::uint8_t opcode) {
    return opcode >= static_cast<std::uint8_t>(AIOpcode::AGENT_CALL);
}

}  // namespace

bool VM::Execute(const std::vector<std::uint8_t>& bytecode, ExecutionContext& context) {
    if (context.gas_remaining == 0) {
        return false;
    }

    for (; context.pc < bytecode.size(); ++context.pc) {
        const std::uint8_t opcode = bytecode[context.pc];
        if (context.gas_remaining < 1) {
            return false;
        }

        if (IsAIInstruction(opcode)) {
            context.gas_remaining -= GasCostForOpcode(static_cast<AIOpcode>(opcode));
            context.trace.emplace_back("ai-opcode");
            continue;
        }

        context.gas_remaining -= 1;
        context.trace.emplace_back("opcode");
    }

    return true;
}

}  // namespace ai_vm
