package types

// V327TextualFeatures are the v3.2.7 switches for SIGN_MODE_TEXTUAL with the PQC extension.
//
// Empty on purpose. The chain already accepts SIGN_MODE_TEXTUAL: the auth tx
// module enables it whenever the bank keeper is wired (it is), so the ante
// handler's accepted sign modes do not change in v3.2.7. The textual work of
// this release is client side (`tx pqc cosign --sign-mode textual`) and
// tests; see testnet/handoff-v327/textual-findings.md.
var V327TextualFeatures = []string{}
