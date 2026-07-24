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
            const auto ai = static_cast<AIOpcode>(opcode);
            const std::uint64_t cost = GasCostForOpcode(ai);
            if (context.gas_remaining < cost) {
                context.trace.emplace_back("out-of-gas:" + OpcodeName(ai));
                return false;
            }
            context.gas_remaining -= cost;
            if (context.handler) {
                if (!context.handler(ai, context)) {
                    context.trace.emplace_back("handler-failed:" + OpcodeName(ai));
                    return false;
                }
            } else {
                context.trace.emplace_back("ai-opcode:" + OpcodeName(ai));
            }
            continue;
        }

        context.gas_remaining -= 1;
        context.trace.emplace_back("opcode");
    }

    return true;
}

}  // namespace ai_vm
