package types

// Event types emitted by the AMM module.
const (
	EventTypePoolCreated       = "amm_pool_created"
	EventTypeLiquidityAdded    = "amm_liquidity_added"
	EventTypeLiquidityRemoved  = "amm_liquidity_removed"
	EventTypeSwap              = "amm_swap"
	EventTypePoolPaused        = "amm_pool_paused"
	EventTypePoolResumed       = "amm_pool_resumed"
	EventTypeParamsUpdated     = "amm_params_updated"
	EventTypeWeightedPriceTick = "amm_weighted_price_tick"
)

// Event attribute keys.
const (
	AttrKeyPoolID        = "pool_id"
	AttrKeyPoolType      = "pool_type"
	AttrKeySender        = "sender"
	AttrKeyTokenA        = "token_a"
	AttrKeyTokenB        = "token_b"
	AttrKeyTokensIn      = "tokens_in"
	AttrKeyTokensOut     = "tokens_out"
	AttrKeyAmountIn      = "amount_in"
	AttrKeyAmountOut     = "amount_out"
	AttrKeyDenomIn       = "denom_in"
	AttrKeyDenomOut      = "denom_out"
	AttrKeyFeeBps        = "fee_bps"
	AttrKeyProtocolFee   = "protocol_fee"
	AttrKeyLPMinted      = "lp_minted"
	AttrKeyLPBurned      = "lp_burned"
	AttrKeyLPDenom       = "lp_denom"
	AttrKeyReserveA      = "reserve_a"
	AttrKeyReserveB      = "reserve_b"
	AttrKeyWeightedPrice = "weighted_price"
	AttrKeyAuthority     = "authority"
	AttrKeyReason        = "reason"
)
