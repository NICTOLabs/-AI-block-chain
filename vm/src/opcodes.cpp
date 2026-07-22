#include "opcodes.h"

namespace ai_vm {

std::string OpcodeName(AIOpcode opcode) {
    switch (opcode) {
    case AIOpcode::AGENT_CALL:
        return "AGENT_CALL";
    case AIOpcode::MODEL_QUERY:
        return "MODEL_QUERY";
    case AIOpcode::PAY_COMPUTE:
        return "PAY_COMPUTE";
    case AIOpcode::VERIFY_OUTPUT:
        return "VERIFY_OUTPUT";
    case AIOpcode::APIKEY_GET:
        return "APIKEY_GET";
    case AIOpcode::AGENT_DELEGATE:
        return "AGENT_DELEGATE";
    }
    return "UNKNOWN";
}

std::uint64_t GasCostForOpcode(AIOpcode opcode) {
    switch (opcode) {
    case AIOpcode::AGENT_CALL:
        return 5000;
    case AIOpcode::MODEL_QUERY:
        return 2000;
    case AIOpcode::PAY_COMPUTE:
        return 1500;
    case AIOpcode::VERIFY_OUTPUT:
        return 3000;
    case AIOpcode::APIKEY_GET:
        return 1000;
    case AIOpcode::AGENT_DELEGATE:
        return 2500;
    }
    return 0;
}

}  // namespace ai_vm
