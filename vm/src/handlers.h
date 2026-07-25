#pragma once

#include "opcodes.h"
#include "vm.h"

namespace ai_vm {

bool DefaultAgentCallHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DefaultModelQueryHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DefaultPayComputeHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DefaultVerifyOutputHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DefaultApiKeyGetHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DefaultAgentDelegateHandler(AIOpcode opcode, ExecutionContext& ctx);
bool DispatchAIHandler(AIOpcode opcode, ExecutionContext& ctx);

}  // namespace ai_vm
