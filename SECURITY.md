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

We will acknowledge your report within 48 hours and provide a detailed response within 7 days.

## Scope

The following areas are in scope for security reports:
- Consensus vulnerabilities
- PQC cryptographic implementation issues
- Bridge security (circuit breaker bypass, attestation forgery)
- AI module manipulation (anomaly detection bypass)
- Privilege escalation
- Denial of service

## Disclosure Policy

- We follow responsible disclosure practices
- Reporters will be credited (unless anonymity is requested)
- We aim to fix critical vulnerabilities within 30 days
- Public disclosure after fix is deployed to mainnet

## PQC-Specific Security

QoreChain uses post-quantum cryptographic primitives (Dilithium-5, ML-KEM-1024). If you discover issues related to:
- Side-channel attacks in the PQC implementation
- Parameter weaknesses
- Key management vulnerabilities

Please report these with the highest priority.
