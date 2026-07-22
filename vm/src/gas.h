#pragma once

#include <cstdint>

namespace ai_vm {

/// Return the base gas cost for an ordinary instruction.
std::uint64_t BaseGasCost();

/// Return the gas penalty for an invalid opcode.
std::uint64_t InvalidOpcodeGasCost();

}  // namespace ai_vm
