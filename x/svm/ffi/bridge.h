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

// Deployment (stub — returns error until program store is implemented)
int32_t qore_svm_deploy_program(
    const uint8_t* elf_bytes, size_t elf_len,
    uint8_t* program_id_out);

// Built-in programs for genesis initialization
int32_t qore_svm_get_builtin_programs(
    uint8_t* out, size_t out_cap, size_t* out_len);

#endif
