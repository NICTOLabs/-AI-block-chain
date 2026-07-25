#include "vm.h"

#include <algorithm>
#include <cstring>
#include <stdexcept>

#include "opcodes.h"
#include "handlers.h"

namespace ai_vm {

namespace {

bool IsAIInstruction(std::uint8_t opcode) {
    return opcode >= static_cast<std::uint8_t>(AIOpcode::AGENT_CALL);
}

bool DefaultAIHandler(AIOpcode opcode, ExecutionContext& context) {
    switch (opcode) {
    case AIOpcode::AGENT_CALL: {
        const char* result = "agent_called";
        context.memory.insert(context.memory.end(), result, result + std::strlen(result));
        context.trace.emplace_back("agent-call:dispatched");
        return true;
    }
    case AIOpcode::MODEL_QUERY: {
        const char* result = "model_result:ok";
        context.memory.insert(context.memory.end(), result, result + std::strlen(result));
        context.trace.emplace_back("model-query:executed");
        return true;
    }
    case AIOpcode::PAY_COMPUTE: {
        context.trace.emplace_back("pay-compute:settled");
        return true;
    }
    case AIOpcode::VERIFY_OUTPUT: {
        context.trace.emplace_back("verify-output:verified");
        return true;
    }
    case AIOpcode::APIKEY_GET: {
        const char* result = "apikey:granted";
        context.memory.insert(context.memory.end(), result, result + std::strlen(result));
        context.trace.emplace_back("apikey-get:issued");
        return true;
    }
    case AIOpcode::AGENT_DELEGATE: {
        context.trace.emplace_back("agent-delegate:dispatched");
        return true;
    }
    }
    context.trace.emplace_back("unknown-ai-opcode");
    return false;
}

}  // namespace

bool VM::Execute(const std::vector<std::uint8_t>& bytecode, ExecutionContext& context) {
    if (context.gas_remaining == 0) {
        return false;
    }

    if (!context.handler) {
        context.handler = DefaultAIHandler;
    }

    for (; context.pc < bytecode.size(); ++context.pc) {
        const std::uint8_t opcode = bytecode[context.pc];
        if (context.gas_remaining < 1) {
            context.trace.emplace_back("out-of-gas:base");
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
            bool handled = false;
            if (context.handler) {
                handled = context.handler(ai, context);
            } else {
                handled = DispatchAIHandler(ai, context);
            }
            if (!handled) {
                context.trace.emplace_back("handler-failed:" + OpcodeName(ai));
                return false;
            }
            continue;
        }

        context.gas_remaining -= 1;
        context.trace.emplace_back("opcode");
    }

    return true;
}

}  // namespace ai_vm
