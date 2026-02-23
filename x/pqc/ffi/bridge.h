#ifndef QORE_PQC_BRIDGE_H
#define QORE_PQC_BRIDGE_H

#include <stdint.h>
#include <stddef.h>

/* ---- Legacy algorithm-specific exports (backward compatibility) ---- */

// Dilithium-5
int32_t qore_dilithium_keygen(uint8_t* pubkey_out, size_t* pubkey_len, uint8_t* privkey_out, size_t* privkey_len);
int32_t qore_dilithium_sign(const uint8_t* privkey, size_t privkey_len, const uint8_t* message, size_t message_len, uint8_t* sig_out, size_t* sig_len);
int32_t qore_dilithium_verify(const uint8_t* pubkey, size_t pubkey_len, const uint8_t* message, size_t message_len, const uint8_t* signature, size_t sig_len);

// ML-KEM-1024
int32_t qore_mlkem_keygen(uint8_t* pubkey_out, size_t* pubkey_len, uint8_t* privkey_out, size_t* privkey_len);
int32_t qore_mlkem_encapsulate(const uint8_t* pubkey, size_t pubkey_len, uint8_t* ciphertext_out, size_t* ciphertext_len, uint8_t* shared_secret_out, size_t* shared_secret_len);
int32_t qore_mlkem_decapsulate(const uint8_t* privkey, size_t privkey_len, const uint8_t* ciphertext, size_t ciphertext_len, uint8_t* shared_secret_out, size_t* shared_secret_len);

// Random beacon
int32_t qore_random_beacon(const uint8_t* seed, size_t seed_len, uint64_t epoch, uint8_t* output, size_t output_len);

// Info
const char* qore_pqc_version(void);
const char* qore_pqc_algorithms(void);

/* ---- Algorithm-aware exports (v0.6.0 — Algorithm Agility Framework) ---- */

// Algorithm-aware keygen (dispatches based on algorithm_id)
int32_t qore_pqc_keygen(uint32_t algorithm_id, uint8_t* pubkey_out, size_t* pubkey_len, uint8_t* privkey_out, size_t* privkey_len);

// Algorithm-aware sign
int32_t qore_pqc_sign(uint32_t algorithm_id, const uint8_t* privkey, size_t privkey_len, const uint8_t* msg, size_t msg_len, uint8_t* sig_out, size_t* sig_len);

// Algorithm-aware verify (returns 1=valid, 0=invalid, <0=error)
int32_t qore_pqc_verify(uint32_t algorithm_id, const uint8_t* pubkey, size_t pubkey_len, const uint8_t* msg, size_t msg_len, const uint8_t* sig, size_t sig_len);

// Query algorithm info
int32_t qore_pqc_algorithm_info(uint32_t algorithm_id, uint32_t* pubkey_size, uint32_t* privkey_size, uint32_t* output_size);

// List supported algorithms
int32_t qore_pqc_list_algorithms(uint32_t* ids_out, size_t* count);

#endif
