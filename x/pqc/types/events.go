package types

// Event types and attribute keys of x/pqc. Older events of the module name
// their strings inline; new ones are declared here.
const (
	// EventTypeEVMWindowUsed is emitted each time an EVM transaction is
	// admitted against the sender's EVM authorisation window, from
	// FeatureEVMWindowUsedEvent on. It is emitted in the EVM-lane ante
	// handler, so it is part of the transaction's result whether execution
	// then succeeds or reverts, exactly as the consumption itself stands.
	EventTypeEVMWindowUsed = "pqc_evm_window_used"

	// AttributeKeyAddress is the account in bech32 (qor1...), the form the
	// window is keyed and queried by.
	AttributeKeyAddress = "address"
	// AttributeKeyEVMAddress is the same account in 0x form.
	AttributeKeyEVMAddress = "evm_address"
	// AttributeKeyWindowOpenHeight is the height the window was opened at.
	AttributeKeyWindowOpenHeight = "window_open_height"
	// AttributeKeyUsedTxs is the window's used transaction count after this
	// transaction.
	AttributeKeyUsedTxs = "used_txs"
	// AttributeKeyUsedValue is the window's used value after this
	// transaction, in uqor as a decimal string.
	AttributeKeyUsedValue = "used_value"
	// AttributeKeyTxValue is what this transaction took from the window, in
	// uqor as a decimal string: the value it transfers plus its maximum fee
	// (gas limit times fee cap), rounded up from the EVM unit. It is the
	// difference between used_value before and after.
	AttributeKeyTxValue = "tx_value"
)
