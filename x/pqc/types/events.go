package types

// Event types and attribute keys of x/pqc. Older events of the module name
// their strings inline; new ones are declared here.
const (
	// EventTypeEVMWindowUsed is emitted each time an EVM transaction is
	// charged to an EVM authorisation window, from FeatureEVMWindowUsedEvent
	// on, once per charge. AttributeKeySource says which charge it is:
	//   - EVMWindowUsedSourceAnte: the EVM-lane ante handler admitted the
	//     transaction against the sender's window. It is part of the
	//     transaction's result whether execution then succeeds or reverts,
	//     exactly as the consumption itself stands.
	//   - EVMWindowUsedSourceSettlement: after a successful execution the
	//     delegated-actor settlement charged a window (a delegated account
	//     that acted, or the sender's value lost beyond what it declared).
	//     It is in the result only when the execution's state is committed,
	//     again exactly as the charge.
	EventTypeEVMWindowUsed = "pqc_evm_window_used"

	// EVMWindowUsedSourceAnte and EVMWindowUsedSourceSettlement are the
	// values of AttributeKeySource.
	EVMWindowUsedSourceAnte       = "ante"
	EVMWindowUsedSourceSettlement = "settlement"

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
	// AttributeKeyTxValue is what this charge added to the window's used
	// value, in uqor as a decimal string: the difference between used_value
	// before and after. In the ante handler it is the value the transaction
	// transfers plus its maximum fee (gas limit times fee cap); in the
	// settlement it is the value the account lost (beyond the declared value,
	// for the sender). Both are rounded up from the EVM unit.
	AttributeKeyTxValue = "tx_value"
	// AttributeKeySource is which charge the event reports: "ante" or
	// "settlement".
	AttributeKeySource = "source"
)
