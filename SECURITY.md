# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them to: **security@qore.network**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will acknowledge your report within 48 hours and give a triage decision within 21 days for Critical and High and within 60 days for Medium, Low and Insight.

## Scope

The bug bounty scope is published, and kept current, on the security page at
https://qorechain.io/security. At the time of writing it covers:
- Consensus and block production, including staking, slashing and distribution
- The bank module, locked-coin accounting and escrow accounts
- The ML-DSA-87 signing path and post-quantum key registration, Go implementation
- The EVM execution lane
- Public RPC and API infrastructure
- The QoreX wallet

Components under active remediation (such as the CosmWasm lane, the bridge
module and the SVM execution lane) are added to the scope, with an
announcement, as their remediation lands. Volumetric and denial of service
testing against production, social engineering, third party dependencies
without a demonstrated impact on QoreChain, testing on mainnet that affects
real users or funds, and automated mass submissions are out of scope. Every
report needs a working proof of concept against a local devnet or the public
testnet.

## Disclosure Policy

- We follow responsible disclosure practices
- Reporters will be credited (unless anonymity is requested)
- We aim to fix critical vulnerabilities within 90 days
- Public disclosure after fix is deployed to mainnet

## PQC-Specific Security

QoreChain uses post-quantum cryptographic primitives (Dilithium-5, ML-KEM-1024). If you discover issues related to:
- Side-channel attacks in the PQC implementation
- Parameter weaknesses
- Key management vulnerabilities

Please report these with the highest priority.
