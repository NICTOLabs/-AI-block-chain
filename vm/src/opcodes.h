#pragma once

#include <cstdint>
#include <string>

namespace ai_vm {

enum class AIOpcode : std::uint8_t {
    AGENT_CALL = 0xA0,
    MODEL_QUERY = 0xA1,
    PAY_COMPUTE = 0xA2,
    VERIFY_OUTPUT = 0xA3,
    APIKEY_GET = 0xA4,
    AGENT_DELEGATE = 0xA5,
};

/// Return the opcode mnemonic for the supplied instruction.
std::string OpcodeName(AIOpcode opcode);

/// Return the gas cost for the supplied AI opcode.
std::uint64_t GasCostForOpcode(AIOpcode opcode);

}  // namespace ai_vm
