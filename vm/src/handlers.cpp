#include "handlers.h"

#include <algorithm>
#include <iomanip>
#include <random>
#include <sstream>
#include <string>

namespace ai_vm {

namespace {

std::string ReadString(ExecutionContext& ctx, std::size_t offset, std::size_t length) {
    if (offset + length > ctx.memory.size()) {
        return {};
    }
    return std::string(reinterpret_cast<const char*>(ctx.memory.data() + offset), length);
}

void WriteString(ExecutionContext& ctx, std::string value) {
    ctx.memory.insert(ctx.memory.end(), value.begin(), value.end());
    ctx.memory.push_back(0);
}

std::string ChecksumOf(const std::string& data) {
    std::uint64_t h1 = 1469598103934665603ULL;
    std::uint64_t h2 = 0;
    for (unsigned char c : data) {
        h1 ^= c;
        h1 *= 1099511628211ULL;
        h2 += h1;
    }
    h1 ^= h2;
    std::ostringstream oss;
    oss << std::hex << std::setfill('0') << std::setw(16) << h1 << std::setw(16) << h2;
    return oss.str();
}

}  // namespace

bool DefaultAgentCallHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:agent_call");
    return true;
}

bool DefaultModelQueryHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:model_query");
    std::string synthetic = "model_response:" + ReadString(ctx, 0, std::min<std::size_t>(32, ctx.memory.size()));
    WriteString(ctx, synthetic);
    return true;
}

bool DefaultPayComputeHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:pay_compute");
    return true;
}

bool DefaultVerifyOutputHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:verify_output");
    std::string payload = ReadString(ctx, 0, std::min<std::size_t>(64, ctx.memory.size()));
    std::string expected = ChecksumOf(payload);
    WriteString(ctx, "verified:" + expected);
    return true;
}

bool DefaultApiKeyGetHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:apikey_get");
    WriteString(ctx, "apikey:tender-demo-key-0001");
    return true;
}

bool DefaultAgentDelegateHandler(AIOpcode opcode, ExecutionContext& ctx) {
    ctx.trace.emplace_back("handler:agent_delegate");
    std::string delegate = ReadString(ctx, 0, std::min<std::size_t>(32, ctx.memory.size()));
    WriteString(ctx, "delegate:" + delegate);
    return true;
}

bool DispatchAIHandler(AIOpcode opcode, ExecutionContext& ctx) {
    switch (opcode) {
        case AIOpcode::AGENT_CALL:
            return DefaultAgentCallHandler(opcode, ctx);
        case AIOpcode::MODEL_QUERY:
            return DefaultModelQueryHandler(opcode, ctx);
        case AIOpcode::PAY_COMPUTE:
            return DefaultPayComputeHandler(opcode, ctx);
        case AIOpcode::VERIFY_OUTPUT:
            return DefaultVerifyOutputHandler(opcode, ctx);
        case AIOpcode::APIKEY_GET:
            return DefaultApiKeyGetHandler(opcode, ctx);
        case AIOpcode::AGENT_DELEGATE:
            return DefaultAgentDelegateHandler(opcode, ctx);
    }
    ctx.trace.emplace_back("unknown-ai-opcode");
    return false;
}

}  // namespace ai_vm
