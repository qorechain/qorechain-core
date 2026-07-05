package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper is the minimal x/bank surface the module needs to move native QOR
// FROM a canonical account on the Native lane (MsgExecuteCosmos, v3.1.85). The
// send is executed under module authority once the authenticator's signature +
// "send" permission + SpendingRule have been verified — the account's own
// signature is not required (that is the point of a linked authenticator). The
// signature matches cosmos-sdk bankkeeper.BaseKeeper.
type BankKeeper interface {
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
}
