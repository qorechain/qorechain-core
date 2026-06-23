#ifndef QORE_SVM_BRIDGE_H
#define QORE_SVM_BRIDGE_H

#include <stdint.h>
#include <stddef.h>

// Version
const char* qore_svm_version(void);

// Lifecycle
void* qore_svm_init(uint64_t compute_budget);
void qore_svm_free(void* handle);

// Validation
int32_t qore_svm_validate_elf(const uint8_t* elf_bytes, size_t elf_len);

// Execution — input_data is mutable (program may modify the input region).
int32_t qore_svm_execute(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* input_data, size_t input_len,
    uint64_t compute_budget,
    uint8_t* result_out, size_t* result_cap);

// Execution with a CPI program registry (programs: count[u64] then
// count*{id[32]|elf_len[u64]|elf_bytes}) enabling BPF->BPF cross-program calls.
int32_t qore_svm_execute_with_programs(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* input_data, size_t input_len,
    const uint8_t* programs, size_t programs_len,
    uint64_t compute_budget,
    uint8_t* result_out, size_t* result_cap);

// Deployment (stub — returns error until program store is implemented)
int32_t qore_svm_deploy_program(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* program_id_out);

// Built-in programs for genesis initialization
int32_t qore_svm_get_builtin_programs(
    uint8_t* out, size_t out_cap, size_t* out_len);

// V2 execution with full account context
int32_t qore_svm_execute_v2(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* input_data, size_t input_len,
    uint64_t compute_budget,
    int64_t block_time,
    uint8_t* result_out, size_t result_cap,
    size_t* result_len,
    void* callback_ctx,
    int32_t (*sysvar_callback)(void* ctx, uint32_t sysvar_id,
                                uint8_t* out, size_t out_cap, size_t* out_len));

// V2 execution with full account context plus a CPI program registry
// (programs: count[u64] then count*{id[32]|elf_len[u64]|elf_bytes}) enabling
// BPF->BPF cross-program invocation from the Solana-compatible execution path.
int32_t qore_svm_execute_v2_with_programs(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* input_data, size_t input_len,
    const uint8_t* programs, size_t programs_len,
    uint64_t compute_budget,
    int64_t block_time,
    uint8_t* result_out, size_t result_cap,
    size_t* result_len);

// Native program execution (no BPF)
int32_t qore_svm_execute_native(
    const uint8_t* program_id,
    uint8_t* input_data, size_t input_len,
    int64_t block_time,
    uint8_t* result_out, size_t result_cap,
    size_t* result_len);

#endif
