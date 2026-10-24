package types

// x/babylon has no transaction messages.
//
// MsgSubmitBTCCheckpoint and MsgBTCRestake used to be declared here as plain Go
// structs. They were never routable: they are not protobuf messages, the module
// registers no interfaces and no Msg service (module.go), so no transaction
// could carry them and no block ever executed one. They were removed in v3.2.5
// (audit 2026-10-25, BAB-01) so that nobody wires them by mistake: a restake
// message in that shape carries no proof of the Bitcoin transaction it names,
// and routing it would credit restaking positions on an unverified claim.
//
// A future restaking message must be a protobuf Msg with a deliberate Msg
// service, and must verify an SPV proof of the Bitcoin transaction (block
// header chain with work, Merkle inclusion, confirmations) before it records a
// position. Until then positions and checkpoints exist only as genesis puts
// them there.
