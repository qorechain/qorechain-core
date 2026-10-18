package types

// Event types and attribute keys of the stakingtier module.
const (
	EventTypeLockCreated     = "staking_tier_lock_created"
	EventTypeLockUnlockEarly = "staking_tier_lock_unlocked_early"
	EventTypeLockReleased    = "staking_tier_lock_released"
	EventTypeBonusClaimed    = "staking_tier_bonus_claimed"
	EventTypeBonusSettled    = "staking_tier_bonus_settled"
	EventTypeParamsUpdated   = "staking_tier_params_updated"

	AttributeKeyLockID        = "lock_id"
	AttributeKeyDelegator     = "delegator"
	AttributeKeyValidator     = "validator"
	AttributeKeyAmount        = "amount"
	AttributeKeyMonths        = "months"
	AttributeKeyEndTime       = "end_time"
	AttributeKeyPenalty       = "penalty_burned"
	AttributeKeyForfeited     = "bonus_forfeited"
	AttributeKeyOwed          = "owed"
	AttributeKeyFunded        = "funded"
	AttributeKeyMinted        = "minted"
	AttributeKeyBaseAPYBps    = "base_apy_bps"
	AttributeKeyPeriodSeconds = "period_seconds"
)
