#include <cassert>
#include <iostream>

#include "opcodes.h"
#include "vm.h"

int main() {
    ai_vm::ExecutionContext context;
    context.gas_remaining = 10'000;
    const std::vector<std::uint8_t> bytecode = {
        static_cast<std::uint8_t>(ai_vm::AIOpcode::AGENT_CALL),
        static_cast<std::uint8_t>(ai_vm::AIOpcode::MODEL_QUERY),
        static_cast<std::uint8_t>(ai_vm::AIOpcode::PAY_COMPUTE),
    };
    const bool ok = ai_vm::VM::Execute(bytecode, context);
    assert(ok);
    std::cout << "vm tests passed" << std::endl;
    return 0;
}
