package types

const (
	EventTypeBTCRestake      = "btc_restake"
	EventTypeBTCUnbond       = "btc_unbond"
	EventTypeCheckpoint      = "btc_checkpoint"
	EventTypeEpochComplete   = "babylon_epoch_complete"

	AttributeKeyStaker       = "staker"
	AttributeKeyBTCTxHash    = "btc_tx_hash"
	AttributeKeyAmount       = "amount"
	AttributeKeyEpoch        = "epoch"
	AttributeKeyCheckpointID = "checkpoint_id"
	AttributeKeyValidator    = "validator"
)
