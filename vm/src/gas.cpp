#include "gas.h"

namespace ai_vm {

std::uint64_t BaseGasCost() {
    return 1;
}

std::uint64_t InvalidOpcodeGasCost() {
    return 100;
}

}  // namespace ai_vm
