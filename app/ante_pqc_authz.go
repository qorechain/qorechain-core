package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosante "github.com/cosmos/evm/ante/cosmos"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// pqcKeyAuthzLimiter refuses an account's own post-quantum key messages
// (register, rotate, migrate) inside authz MsgExec, nested to any depth, from
// FeaturePQCRotationReplayGuard. A grant let a grantee replay a rotation the
// account had signed before a key cycle (QSR-2026-0521, 0506). Below the
// switch the decorator is skipped, so earlier blocks replay unchanged.
func pqcKeyAuthzLimiter(pqc pqcmod.PQCKeeper) featureGatedDecorators {
	return featureGatedDecorators{
		pqc:     pqc,
		feature: pqctypes.FeaturePQCRotationReplayGuard,
		inner:   []sdk.AnteDecorator{cosmosante.NewAuthzLimiterDecorator(pqctypes.PQCKeyManagementMsgTypeURLs()...)},
	}
}
